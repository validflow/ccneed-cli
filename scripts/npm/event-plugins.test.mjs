import assert from 'node:assert/strict';
import test from 'node:test';
import { mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import openclaw, { registerOpenClaw } from '../../plugins/openclaw.mjs';
import registerPi from '../../plugins/pi.mjs';

function fixture() {
  const cwd = mkdtempSync(join(tmpdir(), 'ccneed-events-'));
  const plan = { status: 'ready', plan_token: 'fixture-token', expires_at: '2099-01-01T00:00:00Z', research_requirement: '了解品牌反馈', platforms: [{ platform: 'weibo', quota: 50, entity_keywords: ['测试品牌'], other_keywords: ['用户反馈'] }], estimated_credits: 50, balance: 100 };
  const planFile = join(cwd, 'prepared.json');
  writeFileSync(planFile, JSON.stringify(plan));
  const command = 'ccneed kb start --plan-file "' + planFile.replaceAll('\\', '/') + '"';
  const handlers = new Map();
  const api = { on: (event, handler) => handlers.set(event, handler), runtime: { version: '2026.11.4' } };
  return { cwd, plan, planFile, command, handlers, api, options: { stateDir: join(cwd, 'state') } };
}

test('OpenClaw exposes a native plugin manifest and event registration', () => {
  const data = fixture();
  assert.equal(openclaw.id, 'ccneed');
  assert.equal(typeof openclaw.register, 'function');
  const manifest = JSON.parse(readFileSync(new URL('../../openclaw.plugin.json', import.meta.url)));
  assert.equal(manifest.id, 'ccneed');
  assert.deepEqual(manifest.skills, ['./skills']);
  registerOpenClaw(data.api, data.options);
  assert.ok(data.handlers.has('before_tool_call'));
  assert.ok(data.handlers.has('tool_result_persist'));
});

test('OpenClaw binds persisted plans to one invocation and requests native allow-once approval', () => {
  const data = fixture();
  registerOpenClaw(data.api, data.options);
  const before = data.handlers.get('before_tool_call');
  const persist = data.handlers.get('tool_result_persist');
  const ctx = { sessionKey: 'one-session', workspaceDir: data.cwd };
  const planCall = { toolName: 'exec', toolCallId: 'plan-call', params: { command: 'ccneed kb plan --file draft.json', workdir: data.cwd } };
  assert.equal(before(planCall, ctx), undefined);
  const original = { role: 'toolResult', toolCallId: 'plan-call', content: [{ type: 'text', text: JSON.stringify(data.plan) }], details: { exitCode: 0 } };
  const result = persist({ toolName: 'exec', toolCallId: 'plan-call', message: original }, ctx);
  assert.deepEqual(result.message.content[0], original.content[0]);
  assert.deepEqual(result.message.details, original.details);
  assert.match(result.message.content.at(-1).text, /实体词/);
  assert.match(result.message.content.at(-1).text, /测试品牌/);
  assert.equal(persist({ toolName: 'exec', toolCallId: 'plan-call', message: original }, ctx), undefined);
  const approved = before({ toolName: 'exec', toolCallId: 'start-call', params: { command: data.command, workdir: data.cwd } }, ctx);
  assert.deepEqual(approved.requireApproval.allowedDecisions, ['allow-once', 'deny']);
  assert.match(approved.requireApproval.description, /50/);
  const other = before({ toolName: 'exec', toolCallId: 'other-start', params: { command: data.command, workdir: data.cwd } }, { ...ctx, sessionKey: 'another-session' });
  assert.equal(other.block, true);
  assert.equal(other.requireApproval, undefined);
});

test('OpenClaw preserves unrelated calls and refuses missing identity or an unsupported runtime', () => {
  const data = fixture();
  registerOpenClaw(data.api, data.options);
  const before = data.handlers.get('before_tool_call');
  assert.equal(before({ toolName: 'exec', params: { command: 'rg ccneed README.md' } }, {}), undefined);
  assert.equal(before({ toolName: 'web_fetch', params: { command: data.command } }, {}), undefined);
  assert.equal(before({ toolName: 'exec', params: { command: data.command } }, {}).block, true);
  const old = fixture();
  registerOpenClaw({ ...old.api, runtime: { version: '2026.2.19-2' } }, old.options);
  assert.equal(old.handlers.get('before_tool_call')({ toolName: 'exec', params: { command: 'ccneed account balance' } }, {}), undefined);
  const oldCtx = { sessionKey: 'old-session', workspaceDir: old.cwd };
  old.handlers.get('before_tool_call')({ toolName: 'exec', toolCallId: 'old-plan', params: { command: 'ccneed kb plan --file draft.json' } }, oldCtx);
  old.handlers.get('tool_result_persist')({ toolName: 'exec', toolCallId: 'old-plan', message: { content: [{ type: 'text', text: JSON.stringify(old.plan) }] } }, oldCtx);
  const oldStart = old.handlers.get('before_tool_call')({ toolName: 'exec', toolCallId: 'old', params: { command: old.command } }, oldCtx);
  assert.equal(oldStart.block, true);
  assert.match(oldStart.blockReason, /2026\.11\.4/);
});

test('OpenClaw keeps overlapping plan results separated and invalidates a failed refresh', () => {
  const data = fixture();
  registerOpenClaw(data.api, data.options);
  const before = data.handlers.get('before_tool_call');
  const persist = data.handlers.get('tool_result_persist');
  const first = { sessionKey: 'first', workspaceDir: data.cwd };
  const second = { sessionKey: 'second', workspaceDir: data.cwd };
  for (const ctx of [first, second]) before({ toolName: 'exec', toolCallId: 'same-call-id', params: { command: 'ccneed kb plan --file draft.json' } }, ctx);
  const otherPlan = { ...data.plan, research_requirement: 'Another plan' };
  persist({ toolName: 'exec', toolCallId: 'same-call-id', message: { content: [{ type: 'text', text: JSON.stringify(otherPlan) }] } }, second);
  persist({ toolName: 'exec', toolCallId: 'same-call-id', message: { content: [{ type: 'text', text: JSON.stringify(data.plan) }] } }, first);
  const start = { toolName: 'exec', toolCallId: 'start', params: { command: data.command } };
  assert.ok(before(start, first).requireApproval);
  assert.equal(before(start, second).block, true);
  before({ toolName: 'exec', toolCallId: 'failed-refresh', params: { command: 'ccneed kb plan --file draft.json' } }, first);
  persist({ toolName: 'exec', toolCallId: 'failed-refresh', message: { content: [{ type: 'text', text: 'failed' }], isError: true } }, first);
  assert.equal(before(start, first).block, true);
});

test('Pi delivers plan guidance in its native tool result and uses its confirmation UI', async () => {
  const data = fixture();
  const prompts = [];
  const ctx = { cwd: data.cwd, sessionManager: { getSessionId: () => 'pi-session' }, hasUI: true, ui: { confirm: async (...args) => { prompts.push(args); return true; } } };
  registerPi(data.api, data.options);
  const result = await data.handlers.get('tool_result')({ toolName: 'bash', toolCallId: 'plan', input: { command: 'ccneed kb plan --file draft.json' }, content: [{ type: 'text', text: JSON.stringify(data.plan) }], isError: false }, ctx);
  assert.match(result.content.at(-1).text, /用户反馈/);
  assert.equal(result.content.length, 2);
  const call = { toolName: 'bash', toolCallId: 'start', input: { command: data.command } };
  assert.equal(await data.handlers.get('tool_call')(call, ctx), undefined);
  assert.equal(prompts.length, 1);
  assert.match(prompts[0][1], /50/);
  const denied = await data.handlers.get('tool_call')(call, { ...ctx, ui: { confirm: async () => false } });
  assert.equal(denied.block, true);
  assert.match(denied.reason, /取消/);
  const headless = await data.handlers.get('tool_call')(call, { ...ctx, hasUI: false });
  assert.equal(headless.block, true);
  assert.equal(prompts.length, 1);
});

test('Pi never prompts for unrelated/read-only tools and blocks a changed or unshown plan', async () => {
  const data = fixture();
  registerPi(data.api, data.options);
  const ctx = { cwd: data.cwd, sessionManager: { getSessionId: () => 'pi-session' }, hasUI: true, ui: { confirm: async () => { assert.fail('No approval for invalid or read-only commands'); } } };
  const before = data.handlers.get('tool_call');
  assert.equal(await before({ toolName: 'read', input: { command: data.command } }, ctx), undefined);
  assert.equal(await before({ toolName: 'bash', input: { command: 'ccneed account balance' } }, ctx), undefined);
  assert.equal((await before({ toolName: 'bash', input: { command: data.command } }, ctx)).block, true);
  await data.handlers.get('tool_result')({ toolName: 'bash', input: { command: 'ccneed kb plan --file draft.json' }, content: [{ type: 'text', text: JSON.stringify(data.plan) }], isError: false }, ctx);
  writeFileSync(data.planFile, JSON.stringify({ ...data.plan, estimated_credits: 500 }));
  assert.equal((await before({ toolName: 'bash', input: { command: data.command } }, ctx)).block, true);
  const startup = await data.handlers.get('before_agent_start')({}, ctx);
  assert.equal(startup.message.display, false);
  assert.match(startup.message.content, /实体词/);
});

for (const host of ['openclaw', 'pi']) {
  test(host + ' maps cloud HITL approval to native UI while preserving feedback', async () => {
    const data = fixture();
    const action = { thread_id: 'thread', turn_id: 'turn', pending_action: { action_id: 'action', suggested: { ...data.plan, mode: 'add', kb_id: 7, quotas: { weibo: 50 } } } };
    const answer = join(data.cwd, 'answer.json');
    writeFileSync(answer, JSON.stringify({ decision: 'approved' }));
    const command = 'ccneed tasks reply --thread-id thread --turn-id turn --action-id action --answer-file "' + answer.replaceAll('\\', '/') + '"';
    let prompts = 0;
    const ctx = host === 'pi' ? { cwd: data.cwd, sessionManager: { getSessionId: () => 'hitl' }, hasUI: true, ui: { confirm: async () => { prompts++; return true; } } } : { sessionKey: 'hitl', workspaceDir: data.cwd };
    if (host === 'pi') {
      registerPi(data.api, data.options);
      const result = data.handlers.get('tool_result')({ toolName: 'bash', input: { command: 'ccneed tasks watch --thread-id thread --turn-id turn' }, content: [{ type: 'text', text: JSON.stringify(action) }] }, ctx);
      assert.match(result.content.at(-1).text, /测试品牌/);
      assert.equal(await data.handlers.get('tool_call')({ toolName: 'bash', input: { command } }, ctx), undefined);
      assert.equal(prompts, 1);
      writeFileSync(answer, JSON.stringify({ decision: 'approved', response_kind: 'user_feedback', user_reply: '减少条数' }));
      assert.equal(await data.handlers.get('tool_call')({ toolName: 'bash', input: { command } }, ctx), undefined);
      assert.equal(prompts, 1);
    } else {
      registerOpenClaw(data.api, data.options);
      data.handlers.get('before_tool_call')({ toolName: 'exec', toolCallId: 'watch', params: { command: 'ccneed tasks watch --thread-id thread --turn-id turn' } }, ctx);
      const result = data.handlers.get('tool_result_persist')({ toolName: 'exec', toolCallId: 'watch', message: { content: [{ type: 'text', text: JSON.stringify(action) }] } }, ctx);
      assert.match(result.message.content.at(-1).text, /测试品牌/);
      assert.ok(data.handlers.get('before_tool_call')({ toolName: 'exec', toolCallId: 'reply', params: { command } }, ctx).requireApproval);
      writeFileSync(answer, JSON.stringify({ decision: 'approved', response_kind: 'user_feedback', user_reply: '减少条数' }));
      assert.equal(data.handlers.get('before_tool_call')({ toolName: 'exec', toolCallId: 'feedback', params: { command } }, ctx), undefined);
    }
  });
}
