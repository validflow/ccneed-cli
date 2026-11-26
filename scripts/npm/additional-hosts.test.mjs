import assert from 'node:assert/strict';
import test from 'node:test';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { parseAgents } from './skills.mjs';
import { handleHook } from '../../hooks/acquisition.mjs';
import { additionalPluginInstructions } from './additional-plugins.mjs';

const additional = ['openclaw', 'pi', 'zcode', 'qoder', 'trae', 'dsh'];

test('Qoder instructions do not request new permission rules', () => {
  const text = additionalPluginInstructions('qoder', 'C:/package', '0.1.18').join('\n');
  assert.doesNotMatch(text, /permissions\.ask/);
});
test('explicit host selection accepts native plugin hosts without selecting others', () => {
  assert.deepEqual(parseAgents(additional.join(',')), additional);
  assert.deepEqual(parseAgents('pi,codex,pi'), ['pi', 'codex']);
  for (const value of ['pi,', 'all', 'native', 'PI']) assert.throws(() => parseAgents(value));
});
for (const host of additional) test(host + ' provides plan context without intercepting paid commands', () => {
  const plan = { status: 'ready', research_requirement: '比较品牌', platforms: [{ platform: 'weibo', quota: 30, entity_keywords: ['Brand'], other_keywords: ['feedback'] }], estimated_credits: 30, balance: 100 };
  const output = handleHook(host, { hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file plan.json' }, tool_response: JSON.stringify(plan) });
  assert.match(output.hookSpecificOutput.additionalContext, /Brand/);
  assert.match(output.hookSpecificOutput.additionalContext, /feedback/);
  assert.deepEqual(handleHook(host, { hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed kb start --plan-file plan.json' } }), {});
  assert.deepEqual(handleHook(host, { hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed tasks reply --answer-file answer.json' } }), {});
});
