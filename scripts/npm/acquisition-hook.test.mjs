import assert from 'node:assert/strict';
import test from 'node:test';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { handleHook, renderPlan } from '../../hooks/acquisition.mjs';

function fixture() {
  const directory = mkdtempSync(join(tmpdir(), 'ccneed-hook-'));
  const plan = { status: 'ready', plan_token: 'server-token', plan_version: 'v1', expires_at: '2099-11-06T00:00:00Z', research_requirement: '比较品牌反馈', platforms: [{ platform: 'twitter', quota: 300, entity_keywords: ['Brand'], other_keywords: ['feedback'] }], search_plan: [{ platform: 'twitter', keyword: 'Brand', tier: 1, quota: 240, strategy_rationale: 'entity anchor' }, { platform: 'twitter', keyword: 'feedback', tier: 2, quota: 60, strategy_rationale: 'expanded' }], total_quota: 300, estimated_credits: 300, balance: 1000 };
  const filename = join(directory, 'prepared.json');
  writeFileSync(filename, JSON.stringify(plan));
  const input = { session_id: 'same-session', cwd: directory, tool_name: 'Bash', permission_mode: 'default', approvals_reviewer: 'user', approval_policy: 'on-request' };
  return { directory, plan, filename, input, options: { stateDir: join(directory, 'state') } };
}

const quoted = value => '"' + value.replaceAll('\\', '/') + '"';

test('plan presentation groups whole platform quota and hides internal allocation', () => {
  const { plan } = fixture();
  const text = renderPlan(plan);
  for (const value of ['比较品牌反馈', 'Brand', 'feedback', '300', '1000']) assert.ok(text.includes(value));
  for (const value of ['T1', 'T2', 'tier', '240', '60', 'server-token']) assert.ok(!text.includes(value));
});

for (const host of ['claude', 'codex', 'workbuddy']) {
  test(host + ' requires same-session final presentation before native confirmation', () => {
    const data = fixture();
    const start = { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed kb start --plan-file ' + quoted(data.filename) + ' --request-id 00000000-0000-0000-0000-000000000222' } };
    assert.equal(handleHook(host, start, data.options).hookSpecificOutput.permissionDecision, 'deny');
    const shown = handleHook(host, { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file raw.json' }, tool_response: { stdout: JSON.stringify(data.plan) } }, data.options);
    assert.ok(shown.systemMessage.includes('比较品牌反馈'));
    const allowed = handleHook(host, start, data.options);
    assert.notEqual(allowed.hookSpecificOutput?.permissionDecision, 'deny');
    if (host !== 'codex') assert.equal(allowed.hookSpecificOutput.permissionDecision, 'ask');
    data.plan.platforms[0].quota = 301;
    writeFileSync(data.filename, JSON.stringify(data.plan));
    assert.equal(handleHook(host, start, data.options).hookSpecificOutput.permissionDecision, 'deny');
  });
}

test('known bypass mode cannot perform a paid crawl', () => {
  const data = fixture();
  handleHook('claude', { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file raw.json' }, tool_response: data.plan }, data.options);
  const result = handleHook('claude', { ...data.input, permission_mode: 'bypassPermissions', hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed kb start --plan-file ' + quoted(data.filename) } }, data.options);
  assert.equal(result.hookSpecificOutput.permissionDecision, 'deny');
});

test('noncanonical executable case cannot bypass native prefix rules', () => {
  const data = fixture();
  handleHook('codex', { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file raw.json' }, tool_response: data.plan }, data.options);
  const result = handleHook('codex', { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command: 'CCNEED.CMD kb start --plan-file ' + quoted(data.filename) } }, data.options);
  assert.equal(result.hookSpecificOutput?.permissionDecision, 'deny');
});

test('help text consumed as a file value does not disable confirmation guards', () => {
  const data = fixture();
  for (const command of ['ccneed kb start --plan-file --help --request-id id', 'ccneed --no-update-check kb start --plan-file --help --request-id id', 'ccneed tasks reply --answer-file -h --thread-id thread --turn-id turn --action-id action']) {
    const result = handleHook('codex', { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command } }, data.options);
    assert.equal(result.hookSpecificOutput?.permissionDecision, 'deny');
  }
});

test('WorkBuddy paid PowerShell commands cannot use an unverified sandbox approval path', () => {
  const data = fixture();
  handleHook('workbuddy', { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file raw.json' }, tool_response: data.plan }, data.options);
  const result = handleHook('workbuddy', { ...data.input, tool_name: 'PowerShell', hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed kb start --plan-file ' + quoted(data.filename) } }, data.options);
  assert.equal(result.hookSpecificOutput?.permissionDecision, 'deny');
});

test('cloud HITL accepts the displayed final plan and preserves feedback without starting', () => {
  const data = fixture();
  const suggested = { mode: 'create', kb_name: 'Brand', kb_description: data.plan.research_requirement, keywords: ['original preferred seed'], quotas: { twitter: 300 }, content_scope: 'all', search_plan: data.plan.search_plan, plan_token: 'keyword-token', estimated_credits: 300, balance: 1000 };
  const action = { thread_id: 'thread', turn_id: 'turn', pending_action: { action_id: 'action', action_type: 'tool_approval', suggested } };
  handleHook('workbuddy', { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed tasks watch --thread-id thread --turn-id turn' }, tool_response: { stdout: JSON.stringify(action) } }, data.options);
  const filename = join(data.directory, 'answer.json');
  const input = { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed tasks reply --thread-id thread --turn-id turn --action-id action --answer-file ' + quoted(filename) } };
  writeFileSync(filename, JSON.stringify({ decision: 'approved', modified_input: { keywords: ['Brand'], quotas: suggested.quotas, search_plan: suggested.search_plan, keyword_plan_token: 'keyword-token' } }));
  assert.equal(handleHook('workbuddy', input, data.options).hookSpecificOutput.permissionDecision, 'ask');
  writeFileSync(filename, JSON.stringify({ decision: 'approved', modified_input: { keywords: ['Brand'], quotas: suggested.quotas, search_plan: suggested.search_plan, keyword_plan_token: 'keyword-token', user_reply: 'Focus on product experience' } }));
  assert.equal(handleHook('workbuddy', input, data.options).hookSpecificOutput.permissionDecision, 'ask');
  writeFileSync(filename, JSON.stringify({ decision: 'approved', response_kind: 'user_feedback', reply: '减少到100条' }));
  assert.deepEqual(handleHook('workbuddy', input, data.options), {});
  writeFileSync(filename, JSON.stringify({ decision: 'rejected' }));
  assert.deepEqual(handleHook('workbuddy', input, data.options), {});
});

test('cloud approval refuses an unavailable or insufficient wallet quotation', () => {
  const data = fixture();
  const suggested = { mode: 'create', kb_description: data.plan.research_requirement, quotas: { twitter: 300 }, search_plan: data.plan.search_plan, estimated_credits: 300, balance: null };
  handleHook('workbuddy', { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed tasks get --thread-id thread --turn-id turn' }, tool_response: { thread_id: 'thread', turn_id: 'turn', pending_action: { action_id: 'action', suggested } } }, data.options);
  const filename = join(data.directory, 'answer.json');
  writeFileSync(filename, '{"decision":"approved"}');
  const result = handleHook('workbuddy', { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed tasks reply --thread-id thread --turn-id turn --action-id action --answer-file ' + quoted(filename) } }, data.options);
  assert.equal(result.hookSpecificOutput.permissionDecision, 'deny');
});
