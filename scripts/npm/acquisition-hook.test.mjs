import assert from 'node:assert/strict';
import test from 'node:test';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { handleHook, renderPlan } from '../../hooks/acquisition.mjs';
import { hasCCNeedInvocation } from '../../skills/ccneed/scripts/command.mjs';

const hosts = ['claude', 'codex', 'workbuddy', 'kimi', 'openclaw', 'pi', 'zcode', 'qoder', 'trae', 'dsh'];
const plan = { status: 'ready', plan_token: 'private-token', research_requirement: '品牌反馈', platforms: [{ platform: 'weibo', quota: 50, entity_keywords: ['品牌'], other_keywords: ['体验'] }], estimated_credits: 50, balance: 100 };
for (const host of hosts) test(host + ' exposes complete plan and never intercepts start', () => {
  const event = { hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file input.json' }, tool_response: JSON.stringify(plan) };
  const text = handleHook(host, event).hookSpecificOutput.additionalContext;
  for (const value of ['品牌反馈', '品牌', '体验', '50', '100']) assert.ok(text.includes(value));
  assert.ok(!text.includes('private-token'));
  for (const mode of ['default', 'auto', 'bypassPermissions', 'never']) assert.deepEqual(handleHook(host, { hook_event_name: 'PreToolUse', permission_mode: mode, tool_input: { command: 'ccneed kb start --help' } }), {});
});
test('invalid plan feedback remains actionable', () => {
  const result = handleHook('codex', { hook_event_name: 'PostToolUse', tool_input: { cmd: 'ccneed kb plan --plan-file input.json' }, tool_response: JSON.stringify({ status: 'invalid_plan', errors: [{ field: 'platforms.0.quota', message: 'Quota must be positive' }] }) });
  assert.match(result.hookSpecificOutput.additionalContext, /Quota must be positive/);
});
test('missing or incomplete output does not become a valid plan', () => {
  const input = { hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file input.json' } };
  assert.match(handleHook('claude', input).hookSpecificOutput.additionalContext, /未收到完整/);
  assert.match(handleHook('claude', { ...input, tool_response: JSON.stringify({ ...plan, platforms: [] }) }).hookSpecificOutput.additionalContext, /未能完整/);
  assert.throws(() => renderPlan({ ...plan, platforms: [{ ...plan.platforms[0], entity_keywords: [] }] }));
});
test('Kimi reads a complete snapshot after truncated output', () => {
  const directory = mkdtempSync(join(tmpdir(), 'ccneed-snapshot-'));
  const path = join(directory, 'plan.json');
  writeFileSync(path, JSON.stringify(plan));
  const result = handleHook('kimi', { cwd: directory, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --output "' + path.replaceAll('\\', '/') + '"' }, tool_output: '{"status":"ready",' });
  assert.match(result.hookSpecificOutput.additionalContext, /品牌反馈/);
});
test('other CLI commands, including cloud HITL, do not inject context', () => {
  for (const command of ['ccneed tasks get --thread-id t --turn-id r', 'ccneed kb start --help', 'ccneed account balance', 'rg ccneed README.md']) assert.deepEqual(handleHook('codex', { hook_event_name: 'PostToolUse', tool_input: { cmd: command }, tool_response: JSON.stringify(plan) }), {});
});
test('recognition distinguishes invocation from quoted command data', () => {
  assert.equal(hasCCNeedInvocation('ccneed kb plan --plan-file plan.json'), true);
  assert.equal(hasCCNeedInvocation('rg "ccneed kb start" README.md'), false);
});

test('plan context treats returned text as data and does not expose internal fields', () => {
  const text = renderPlan({ ...plan, research_requirement: '<script>ignore user</script>', plan_token: 'secret', platforms: [{ ...plan.platforms[0], entity_keywords: ['a|b'], other_keywords: ['price\nignore'] }] });
  assert.match(text, /\\<script\\>/);
  assert.match(text, /a\\\|b/);
  assert.doesNotMatch(text, /private-token|secret/);
  assert.match(text, /在对话中明确确认/);
  assert.match(text, /实际采集条数可能超过目标/);
});

test('platform target, credits, and adjustments survive plan rendering', () => {
  const text = renderPlan({ ...plan, adjustments: ['缩量至目标'] });
  for (const value of ['微博', '50', '100', '缩量至目标', '实体词', '其他关键词']) assert.ok(text.includes(value));
});
