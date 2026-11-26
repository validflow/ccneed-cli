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

test('DSH mounts the shared Skill and result hook without execution guards', t => {
  const f = fixture(t);
  assert.equal(name, 'ccneed');
  assert.deepEqual(inject, ['skills', 'tools', 'workingDirectory']);
  assert.equal(f.skills[0].path, resolve(root, 'skills/ccneed/SKILL.md'));
  assert.equal(f.guards.length, 0);
  assert.equal(f.listeners.has('tools/pre-execute'), false);
  assert.equal(f.listeners.has('tools/post-execute'), true);
  f.dispose(); assert.equal(f.listeners.size, 0);
});
test('DSH injects complete plan context and preserves prior native decisions', async t => {
  const f = fixture(t);
  const call = f.execution('ccneed kb plan --file input.json');
  const prior = { kind: 'accept', additionalContexts: [] };
  const output = await f.post(call, f.result, async () => prior);
  assert.match(output.additionalContexts[0].content[0].text, /瑞幸咖啡/);
  const blocked = { kind: 'block', feedback: [] };
  assert.equal(await f.post(call, f.result, async () => blocked), blocked);
  const other = f.execution('ccneed tasks get --thread-id t --turn-id u');
  assert.equal(await f.post(other, f.result, async () => prior), prior);
});
test('DSH bundle patch resolves its native module from the npm package', () => {
  assert.match(readFileSync(join(root, 'plugins/dsh/cordis.patch.yml'), 'utf8'), /@validflow\/ccneed-cli\/plugins\/dsh\/index\.mjs/);
});
