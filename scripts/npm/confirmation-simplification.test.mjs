import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, mkdtempSync, mkdirSync, writeFileSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { handleHook } from '../../skills/ccneed/scripts/acquisition.mjs';
import { preparePluginIntegration, attachPluginIntegration, codexRules } from './plugin.mjs';

test('paid commands are not intercepted by CCNeed in any host mode', () => {
  for (const host of ['codex', 'claude', 'workbuddy', 'kimi', 'openclaw', 'pi', 'dsh', 'qoder', 'trae', 'zcode']) {
    for (const command of ['ccneed kb start --help', 'ccneed kb start --plan-file /tmp/plan.json', 'ccneed tasks reply --answer-file /tmp/answer.json']) {
      assert.deepEqual(handleHook(host, { hook_event_name: 'PreToolUse', permission_mode: 'bypassPermissions', tool_input: { command } }), {}, `${host}: ${command}`);
    }
  }
});

test('native plugin hooks register only startup and result events', () => {
  for (const host of ['claude', 'codex', 'workbuddy', 'qoder', 'trae', 'zcode']) {
    const config = JSON.parse(readFileSync(new URL(`../../hooks/${host}.json`, import.meta.url)));
    assert.deepEqual(Object.keys(config.hooks).sort(), ['PostToolUse', 'SessionStart']);
  }
  const kimi = JSON.parse(readFileSync(new URL('../../.kimi-plugin/plugin.json', import.meta.url)));
  assert.deepEqual(kimi.hooks.map(h => h.event).sort(), ['PostToolUse', 'SessionStart']);
});

test('installer removes only its exact old Codex prompt rules and keeps custom rules', () => {
  const home = mkdtempSync(join(tmpdir(), 'ccneed-rules-'));
  const dir = join(home, '.codex', 'rules');
  mkdirSync(dir, { recursive: true });
  const target = join(dir, 'ccneed.rules');
  writeFileSync(target, codexRules());
  const root = resolve('.');
  attachPluginIntegration(preparePluginIntegration(home, ['codex'], root));
  assert.equal(existsSync(target) && readFileSync(target, 'utf8').includes('decision = "prompt"'), false);
  writeFileSync(target, '# custom rule\n');
  attachPluginIntegration(preparePluginIntegration(home, ['codex'], root));
  assert.equal(readFileSync(target, 'utf8'), '# custom rule\n');
});
