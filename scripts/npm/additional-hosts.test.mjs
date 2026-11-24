import assert from 'node:assert/strict';
import test from 'node:test';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { parseAgents } from './skills.mjs';
import { handleHook } from '../../hooks/acquisition.mjs';
import { additionalPluginInstructions } from './additional-plugins.mjs';

const additional = ['openclaw', 'pi', 'zcode', 'qoder', 'trae', 'dsh'];

test('Qoder installation reports the native permission rules needed for collection prompts', () => {
  const instructions = additionalPluginInstructions('qoder', 'C:/package', '0.1.17').join('\n');
  for (const binary of ['ccneed', 'ccneed.cmd', 'ccneed.exe']) {
    for (const command of ['kb start', 'tasks reply']) assert.ok(instructions.includes(`Bash(${binary} ${command}:*)`));
  }
  assert.match(instructions, /permissions\.ask/);
  assert.match(instructions, /mode/i);
});

test('explicit host selection accepts native plugin hosts without selecting others', () => {
  assert.deepEqual(parseAgents(additional.join(',')), additional);
  assert.deepEqual(parseAgents('pi,codex,pi'), ['pi', 'codex']);
  for (const value of ['pi,', 'all', 'native', 'PI']) assert.throws(() => parseAgents(value));
});

for (const host of additional) test(host + ' shares plan, credit and session validation with existing plugins', () => {
  const cwd = mkdtempSync(join(tmpdir(), 'ccneed-native-host-'));
  const options = { stateDir: join(cwd, 'state') };
  const plan = { status: 'ready', plan_token: 'fixture-token', expires_at: '2099-01-01T00:00:00Z', research_requirement: '比较品牌', platforms: [{ platform: 'weibo', quota: 30, entity_keywords: ['Brand'], other_keywords: ['feedback'] }], estimated_credits: 30, balance: 100 };
  const filename = join(cwd, 'prepared.json');
  writeFileSync(filename, JSON.stringify(plan));
  const input = { cwd, session_id: 'session-a', tool_name: 'Bash', permission_mode: 'default' };
  const start = { ...input, hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed kb start --plan-file "' + filename.replaceAll('\\', '/') + '"' } };
  assert.deepEqual(handleHook(host, { hook_event_name: 'PreToolUse', tool_input: { command: 'rg "ccneed kb start" README.md' } }, options), {});
  assert.equal(handleHook(host, start, options).hookSpecificOutput?.permissionDecision, 'deny');
  const response = handleHook(host, { ...input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file plan.json' }, tool_response: JSON.stringify(plan) }, options);
  assert.match(response.hookSpecificOutput?.additionalContext ?? '', /Brand/);
  assert.match(response.hookSpecificOutput.additionalContext, /feedback/);
  const decision = handleHook(host, start, options).hookSpecificOutput?.permissionDecision;
  assert.notEqual(decision, 'deny');
  if (['zcode', 'qoder', 'trae'].includes(host)) assert.equal(decision, 'ask');
  assert.equal(handleHook(host, { ...start, session_id: 'session-b' }, options).hookSpecificOutput?.permissionDecision, 'deny');
  plan.platforms[0].entity_keywords = [];
  writeFileSync(filename, JSON.stringify(plan));
  assert.equal(handleHook(host, start, options).hookSpecificOutput?.permissionDecision, 'deny');
  const invalid = handleHook(host, { ...input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file plan.json' }, tool_response: JSON.stringify(plan) }, options);
  assert.match(invalid.hookSpecificOutput?.additionalContext ?? '', /未能|未登记|无法/);
});

for (const host of additional) test(host + ' supports cloud HITL feedback and binds paid approval to the shown card', () => {
  const cwd = mkdtempSync(join(tmpdir(), 'ccneed-native-hitl-'));
  const options = { stateDir: join(cwd, 'state') };
  const input = { cwd, session_id: 'session', tool_name: 'Bash' };
  const suggested = { mode: 'create', kb_description: '品牌反馈研究', quotas: { weibo: 50 }, search_plan: [{ platform: 'weibo', keyword: 'Brand', quota: 40, tier: 1, strategy_rationale: 'entity anchor' }, { platform: 'weibo', keyword: 'feedback', quota: 10, tier: 2 }], estimated_credits: 50, balance: 100 };
  const card = { thread_id: 'thread', turn_id: 'turn', pending_action: { action_id: 'action', suggested } };
  const show = () => handleHook(host, { ...input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed tasks get --thread-id thread --turn-id turn' }, tool_response: JSON.stringify(card) }, options);
  const context = show().hookSpecificOutput.additionalContext;
  for (const text of ['品牌反馈研究', 'Brand', 'feedback', '50', '100']) assert.ok(context.includes(text));
  const filename = join(cwd, 'answer.json');
  const approve = () => handleHook(host, { ...input, hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed tasks reply --thread-id thread --turn-id turn --action-id action --answer-file "' + filename.replaceAll('\\', '/') + '"' } }, options);
  writeFileSync(filename, JSON.stringify({ decision: 'approved' }));
  assert.equal(approve().hookSpecificOutput.permissionDecision, 'ask');
  writeFileSync(filename, JSON.stringify({ decision: 'approved', modified_input: { quotas: { weibo: 200 } } }));
  assert.equal(approve().hookSpecificOutput.permissionDecision, 'deny');
  for (const answer of [{ decision: 'rejected' }, { decision: 'approved', response_kind: 'user_feedback', user_reply: '减少条数' }]) {
    writeFileSync(filename, JSON.stringify(answer));
    assert.deepEqual(approve(), {});
  }
  handleHook(host, { ...input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed tasks get --thread-id thread --turn-id turn' }, tool_response: 'incomplete {' }, options);
  writeFileSync(filename, JSON.stringify({ decision: 'approved' }));
  assert.equal(approve().hookSpecificOutput.permissionDecision, 'deny', 'an incomplete refresh invalidates the previous approval candidate');
  suggested.balance = 10;
  show();
  writeFileSync(filename, JSON.stringify({ decision: 'approved' }));
  assert.equal(approve().hookSpecificOutput.permissionDecision, 'deny');
});
