import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync, existsSync, mkdtempSync, cpSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { join, resolve, relative } from 'node:path';
import { tmpdir } from 'node:os';
import { spawnSync } from 'node:child_process';

const root = fileURLToPath(new URL('../../', import.meta.url));
const read = path => JSON.parse(readFileSync(join(root, path), 'utf8'));

for (const [host, directory] of [['claude', '.claude-plugin'], ['codex', '.codex-plugin'], ['workbuddy', '.codebuddy-plugin']]) {
  test(host + ' native manifest loads shared skill and its own guarded hook payload', () => {
    const manifest = read(directory + '/plugin.json');
    assert.equal(manifest.name, 'ccneed');
    assert.equal(manifest.version, read('package.json').version);
    assert.equal(manifest.skills.replace(/\/$/, ''), './skills');
    assert.equal(typeof manifest.hooks, 'string');
    const hookPath = resolve(root, manifest.hooks);
    assert.ok(!relative(root, hookPath).startsWith('..'));
    const config = JSON.parse(readFileSync(hookPath, 'utf8'));
    for (const event of ['SessionStart', 'PreToolUse', 'PostToolUse']) {
      assert.ok(config.hooks[event]?.length, event + ' must load from native plugin');
      for (const entry of config.hooks[event]) for (const hook of entry.hooks) {
        assert.equal(hook.type, 'command');
        assert.match(hook.command, new RegExp('--host ' + host));
        assert.match(hook.command, /PLUGIN_ROOT/);
        assert.ok(!hook.command.includes(root), 'payload must survive native cache relocation');
      }
    }
  });
}

test('native acquisition entry executes from a relocated plugin cache with JSON stdin', () => {
  const destination = mkdtempSync(join(tmpdir(), 'ccneed plugin cache '));
  for (const directory of ['hooks', 'skills']) cpSync(join(root, directory), join(destination, directory), { recursive: true });
  const result = spawnSync(process.execPath, [join(destination, 'hooks', 'acquisition.mjs'), '--host', 'codex'], {
    cwd: destination, input: JSON.stringify({ hook_event_name: 'SessionStart', session_id: 'native-session' }),
    encoding: 'utf8', timeout: 10000, windowsHide: true,
  });
  assert.equal(result.status, 0, result.stderr);
  assert.equal(JSON.parse(result.stdout).hookSpecificOutput.hookEventName, 'SessionStart');
});

test('native marketplaces are distributed with the npm package', () => {
  const files = read('package.json').files;
  for (const directory of ['.claude-plugin', '.codex-plugin', '.codebuddy-plugin', '.agents/plugins']) {
    assert.ok(files.some(pattern => pattern === directory || pattern === directory + '/'), directory);
    assert.ok(existsSync(join(root, directory, directory === '.codex-plugin' ? 'plugin.json' : 'marketplace.json')));
  }
});
