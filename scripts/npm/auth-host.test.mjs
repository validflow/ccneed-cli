import test from 'node:test';
import assert from 'node:assert/strict';
import { handleHook } from '../../skills/ccneed/scripts/acquisition.mjs';

test('native host startup tells the Agent its own login identity', () => {
  for (const host of ['codex', 'claude', 'workbuddy', 'kimi', 'openclaw', 'pi', 'zcode', 'qoder', 'trae', 'dsh']) {
    const result = handleHook(host, { hook_event_name: 'SessionStart' });
    assert.ok(result.hookSpecificOutput.additionalContext.includes(`ccneed auth login --agent ${host}`), host);
  }
});
