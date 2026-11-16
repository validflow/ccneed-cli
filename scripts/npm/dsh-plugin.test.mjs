import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { apply, inject, name } from '../../plugins/dsh/index.mjs';

const root = fileURLToPath(new URL('../../', import.meta.url));

// Native Cordis owns these registration boundaries; all plan validation and
// filesystem access use the real shared handler and real files.
function fixture(t) {
  const stateDir = mkdtempSync(join(tmpdir(), 'ccneed-dsh-'));
  t.after(() => rmSync(stateDir, { recursive: true, force: true }));
  const listeners = new Map(), guards = [], skills = [];
  const disposers = [];
  const ctx = {
    skills: { register(skill) { skills.push(skill); disposers.push(() => skills.splice(skills.indexOf(skill), 1)); } },
    tools: { guard(guard) { guards.push(guard); disposers.push(() => guards.splice(guards.indexOf(guard), 1)); } },
    workingDirectory: { get: () => stateDir },
    on(event, callback) { listeners.set(event, callback); disposers.push(() => listeners.delete(event)); },
  };
  apply(ctx, { stateDir });
  const execution = (command, session = 'session-a', extra = {}) => ({
    name: 'bash', arguments: { command }, agent: { id: session, session: { id: session } },
    signal: new AbortController().signal, ...extra,
  });
  const plan = { status: 'ready', plan_token: 'fixture-token', expires_at: new Date(Date.now() + 60_000).toISOString(), research_requirement: '咖啡品牌讨论', platforms: [{ platform: 'weibo', quota: 50, entity_keywords: ['瑞幸咖啡'], other_keywords: ['新品'] }], estimated_credits: 50, balance: 100 };
  const path = join(stateDir, 'plan.json');
  writeFileSync(path, JSON.stringify(plan));
  const result = { isError: false, value: { kind: 'foreground', stdout: { text: JSON.stringify(plan) } }, content: [{ type: 'text', text: JSON.stringify(plan) }] };
  const pre = (exec, next = async () => ({ kind: 'allow' })) => listeners.get('tools/pre-execute')(exec, next);
  const post = (exec, outcome = result, next = async () => ({ kind: 'accept' })) => listeners.get('tools/post-execute')(exec, outcome, next);
  return { stateDir, path, plan, result, ctx, execution, pre, post, guards, skills, listeners, dispose: () => disposers.reverse().forEach(dispose => dispose()) };
}

test('DSH native bundle mounts one shared Skill and effect-owned hooks', t => {
  const f = fixture(t);
  assert.equal(name, 'ccneed');
  assert.deepEqual(inject, ['skills', 'tools', 'workingDirectory']);
  assert.equal(f.skills.length, 1);
  assert.equal(f.skills[0].name, 'ccneed');
  assert.equal(f.skills[0].path, resolve(root, 'skills/ccneed/SKILL.md'));
  assert.deepEqual(f.skills[0].resourceBase, { kind: 'directory', path: resolve(root, 'skills/ccneed') });
  assert.ok(f.skills[0].content.includes('# CCNeed'));
  assert.ok(!f.skills[0].content.startsWith('---'));
  assert.equal(f.guards.length, 1);
  f.dispose();
  assert.equal(f.skills.length, 0);
  assert.equal(f.listeners.size, 0);
  assert.equal(f.guards.length, 0);
});

test('DSH keeps unrelated tools and metadata mentions outside collection checks', async t => {
  const f = fixture(t);
  const value = { kind: 'deny', reason: 'native policy' };
  assert.equal(await f.pre(f.execution('rg "ccneed kb start" README.md'), async () => value), value);
  assert.deepEqual(await f.pre({ name: 'read', arguments: { command: 'ccneed kb start' }, signal: new AbortController().signal }, async () => value), value);
  assert.deepEqual(f.guards.map(guard => guard(f.execution('echo hello'))), [undefined]);
});

test('DSH injects complete plan context and requires native per-call approval', async t => {
  const f = fixture(t);
  const original = structuredClone(f.result);
  const prior = { kind: 'accept', additionalContexts: [{ role: 'user', id: 'existing', source: { kind: 'fixture' }, content: [] }] };
  const posted = await f.post(f.execution('ccneed kb plan --file input.json'), f.result, async () => prior);
  assert.deepEqual(f.result, original);
  assert.equal(posted.additionalContexts.length, 2);
  const injected = posted.additionalContexts[1];
  assert.equal(injected.source.kind, 'ccneed-acquisition');
  assert.equal(injected.source.form, 'instructions');
  assert.ok(injected.id);
  assert.match(injected.content[0].text, /瑞幸咖啡/);
  assert.match(injected.content[0].text, /新品/);
  const start = f.execution('ccneed kb start --plan-file "' + f.path.replaceAll('\\', '/') + '"');
  const decision = await f.pre(start);
  assert.equal(decision.kind, 'ask');
  assert.match(decision.reason, /50/);
  assert.equal(f.guards[0](start), undefined);
});

test('DSH blocks undisplayed, changed and cross-session plans monotonically', async t => {
  const f = fixture(t);
  const command = 'ccneed kb start --plan-file "' + f.path.replaceAll('\\', '/') + '"';
  assert.equal((await f.pre(f.execution(command))).kind, 'deny');
  await f.post(f.execution('ccneed kb plan --file input.json'));
  assert.equal((await f.pre(f.execution(command, 'session-b'))).kind, 'deny');
  writeFileSync(f.path, JSON.stringify({ ...f.plan, balance: 1 }));
  assert.equal((await f.pre(f.execution(command))).kind, 'deny');
  assert.equal(typeof f.guards[0](f.execution(command)), 'string');
});

test('DSH respects other native denials, blocked results and cancellation', async t => {
  const f = fixture(t);
  await f.post(f.execution('ccneed kb plan --file input.json'));
  const start = f.execution('ccneed kb start --plan-file "' + f.path.replaceAll('\\', '/') + '"');
  const deny = { kind: 'deny', reason: 'another policy' };
  assert.equal(await f.pre(start, async () => deny), deny);
  const blocked = { kind: 'block', feedback: [{ type: 'text', text: 'redacted' }] };
  assert.equal(await f.post(f.execution('ccneed kb plan --file input.json'), f.result, async () => blocked), blocked);
  const controller = new AbortController(); controller.abort();
  assert.deepEqual(await f.pre({ ...start, signal: controller.signal }), { kind: 'cancel' });
});

test('DSH preserves native redaction and refuses truncated plans', async t => {
  const f = fixture(t);
  const command = 'ccneed kb start --plan-file "' + f.path.replaceAll('\\', '/') + '"';
  const planCall = f.execution('ccneed kb plan --file input.json');
  await f.post(planCall);
  assert.equal((await f.pre(f.execution(command))).kind, 'ask');
  const redacted = await f.post(planCall, f.result, async () => ({ kind: 'accept', content: [{ type: 'text', text: '[redacted]' }] }));
  assert.doesNotMatch(redacted.additionalContexts[0].content[0].text, /瑞幸咖啡/);
  assert.equal((await f.pre(f.execution(command))).kind, 'deny');
  const truncated = { ...f.result, value: { stdout: { text: JSON.stringify(f.plan), truncated: true } } };
  await f.post(planCall, truncated);
  assert.equal((await f.pre(f.execution(command))).kind, 'deny');
});

test('DSH cloud HITL requests native approval while feedback stays available', async t => {
  const f = fixture(t);
  const suggested = { mode: 'create', kb_description: f.plan.research_requirement, platforms: f.plan.platforms, quotas: { weibo: 50 }, estimated_credits: 50, balance: 100 };
  const action = { thread_id: 'thread', turn_id: 'turn', pending_action: { action_id: 'action', suggested } };
  const result = { isError: false, value: { stdout: { text: JSON.stringify(action) } }, content: [] };
  const response = await f.post(f.execution('ccneed tasks get --thread-id thread --turn-id turn'), result);
  assert.match(response.additionalContexts[0].content[0].text, /瑞幸咖啡/);
  const path = join(f.stateDir, 'answer.json');
  const command = 'ccneed tasks reply --thread-id thread --turn-id turn --action-id action --answer-file "' + path.replaceAll('\\', '/') + '"';
  writeFileSync(path, JSON.stringify({ decision: 'approved' }));
  assert.equal((await f.pre(f.execution(command))).kind, 'ask');
  writeFileSync(path, JSON.stringify({ decision: 'approved', modified_input: { quotas: { weibo: 200 } } }));
  assert.equal((await f.pre(f.execution(command))).kind, 'deny');
  writeFileSync(path, JSON.stringify({ decision: 'rejected' }));
  assert.deepEqual(await f.pre(f.execution(command)), { kind: 'allow' });
  writeFileSync(path, JSON.stringify({ decision: 'approved', response_kind: 'user_feedback', reply: '改成30条' }));
  assert.deepEqual(await f.pre(f.execution(command)), { kind: 'allow' });
});

test('DSH bundle patch resolves its native module from the npm package', () => {
  const patch = readFileSync(join(root, 'plugins/dsh/cordis.patch.yml'), 'utf8');
  assert.match(patch, /insert:/);
  assert.match(patch, /name: ['"]?@validflow\/ccneed-cli\/plugins\/dsh\/index\.mjs/);
});
