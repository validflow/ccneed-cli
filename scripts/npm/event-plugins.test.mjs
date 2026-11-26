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

test('OpenClaw injects plan output without requesting execution approval', () => {
  const data = fixture(); registerOpenClaw(data.api, data.options);
  const ctx = { sessionKey: 'session', workspaceDir: data.cwd };
  const call = { toolName: 'exec', toolCallId: 'plan', params: { command: 'ccneed kb plan --file input.json' } };
  assert.equal(data.handlers.get('before_tool_call')(call, ctx), undefined);
  const message = { content: [{ type: 'text', text: JSON.stringify(data.plan) }] };
  const result = data.handlers.get('tool_result_persist')({ toolName: 'exec', toolCallId: 'plan', message }, ctx);
  assert.match(result.message.content.at(-1).text, /测试品牌/);
  assert.equal(data.handlers.get('before_tool_call')({ toolName: 'exec', toolCallId: 'start', params: { command: data.command } }, ctx), undefined);
});
test('Pi exposes plan context without a pre-execution confirmation handler', () => {
  const data = fixture(); registerPi(data.api, data.options);
  const ctx = { cwd: data.cwd, sessionManager: { getSessionId: () => 'pi-session' } };
  assert.equal(data.handlers.has('tool_call'), false);
  const output = data.handlers.get('tool_result')({ toolName: 'bash', input: { command: 'ccneed kb plan --file input.json' }, content: [{ type: 'text', text: JSON.stringify(data.plan) }] }, ctx);
  assert.match(output.content.at(-1).text, /用户反馈/);
  assert.match(data.handlers.get('before_agent_start')({}, ctx).message.content, /实体词/);
});
