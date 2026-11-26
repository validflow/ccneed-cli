import assert from 'node:assert/strict';
import test from 'node:test';
import { existsSync, mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';

const root = fileURLToPath(new URL('../../', import.meta.url));
const version = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8')).version;

test('TRAE native payload uses the supported Codex manifest and a dedicated hook host', () => {
  const manifest = JSON.parse(readFileSync(join(root, 'plugins', 'trae', '.codex-plugin', 'plugin.json'), 'utf8'));
  assert.equal(manifest.name, 'ccneed');
  assert.equal(manifest.version, version);
  assert.equal(manifest.skills, './skills');
  assert.equal(manifest.hooks, './hooks/trae.json');
  const hooks = JSON.parse(readFileSync(join(root, 'hooks', 'trae.json'), 'utf8')).hooks;
  assert.deepEqual(Object.keys(hooks), ['SessionStart', 'PostToolUse']);
  for (const [event, groups] of Object.entries(hooks)) {
    assert.equal(groups.length, 1);
    assert.equal(groups[0].hooks.length, 1);
    const hook = groups[0].hooks[0];
    assert.equal(hook.type, 'command');
    assert.match(hook.command, /--host trae$/);
    assert.match(hook.commandWindows, /--host trae$/);
    assert.match(hook.command, /\$\{CLAUDE_PLUGIN_ROOT\}/);
    assert.equal(hook.timeout, 10);
    assert.equal(hook.async, undefined);
    if (event !== 'SessionStart') assert.ok(groups[0].matcher.split('|').includes('exec_command'));
  }
});

for (const host of ['zcode', 'qoder']) {
  test(host + ' native manifest loads the shared Skill and exactly one hook source', () => {
    const manifest = JSON.parse(readFileSync(join(root, '.' + host + '-plugin', 'plugin.json'), 'utf8'));
    assert.equal(manifest.name, 'ccneed');
    assert.equal(manifest.version, version);
    assert.equal(manifest.skills, './skills');
    assert.equal(manifest.hooks, './hooks/' + host + '.json');
    assert.equal(existsSync(join(root, 'hooks', 'hooks.json')), false, 'convention hooks must not duplicate explicit host hooks');
    const hooks = JSON.parse(readFileSync(join(root, manifest.hooks), 'utf8')).hooks;
    assert.deepEqual(Object.keys(hooks), ['SessionStart', 'PostToolUse']);
    for (const [event, groups] of Object.entries(hooks)) {
      assert.equal(groups.length, 1);
      const group = groups[0];
      assert.equal(group.hooks.length, 1);
      assert.equal(group.async, undefined);
      const hook = group.hooks[0];
      assert.equal(hook.command, 'node');
      assert.deepEqual(hook.args, ['${' + host.toUpperCase() + '_PLUGIN_ROOT}/hooks/acquisition.mjs', '--host', host]);
      assert.equal(hook.type, host === 'zcode' ? 'process' : 'command');
      assert.equal(hook.async, undefined, 'paid execution must wait for validation');
      assert.equal(hook.shell, undefined, 'native argv form preserves spaces and Unicode paths');
      assert.equal(host === 'zcode' ? hook.timeoutMs : hook.timeout, host === 'zcode' ? 10000 : 10);
      if (event !== 'SessionStart') assert.ok(group.matcher.split('|').includes('Bash'));
    }
  });

  test(host + ' native argv handler exchanges real stdin/stdout without invoking collection', () => {
    const directory = mkdtempSync(join(tmpdir(), 'ccneed-desktop-'));
    const hook = JSON.parse(readFileSync(join(root, 'hooks', host + '.json'), 'utf8')).hooks.SessionStart[0].hooks[0];
    const args = hook.args.map(value => value.replace('${' + host.toUpperCase() + '_PLUGIN_ROOT}', root));
    const input = { hook_event_name: 'SessionStart', session_id: 'native-session', cwd: directory };
    const child = spawnSync(process.execPath, args, { input: JSON.stringify(input), encoding: 'utf8', env: { ...process.env, TEMP: directory, TMP: directory, TMPDIR: directory }, windowsHide: true });
    assert.equal(child.status, 0, child.stderr);
    const result = JSON.parse(child.stdout);
    assert.equal(result.hookSpecificOutput?.hookEventName, 'SessionStart');
    assert.match(result.hookSpecificOutput.additionalContext, /实体词/);
    assert.match(result.hookSpecificOutput.additionalContext, /积分/);
    assert.equal(child.stderr, '');
  });

  test(host + ' native hook processes preserve plan display, validation and approval boundaries', () => {
    const directory = mkdtempSync(join(tmpdir(), 'ccneed desktop 中文 '));
    const hook = JSON.parse(readFileSync(join(root, 'hooks', host + '.json'), 'utf8')).hooks.PostToolUse[0].hooks[0];
    const args = hook.args.map(value => value.replace('${' + host.toUpperCase() + '_PLUGIN_ROOT}', root));
    const plan = { status: 'ready', plan_token: 'fixture-only', plan_version: 'v1', expires_at: '2099-11-06T00:00:00Z', research_requirement: 'Compare public brand feedback', platforms: [{ platform: 'twitter', quota: 50, entity_keywords: ['Brand'], other_keywords: ['feedback'] }], total_quota: 50, estimated_credits: 50, balance: 100 };
    const filename = join(directory, 'prepared.json');
    writeFileSync(filename, JSON.stringify(plan));
    const start = 'ccneed kb start --plan-file "' + filename.replaceAll('\\', '/') + '"';
    const run = (event, command, extra = {}) => {
      const child = spawnSync(process.execPath, args, { input: JSON.stringify({ hook_event_name: event, session_id: 'native-session', cwd: directory, tool_name: 'Bash', permission_mode: 'default', tool_input: { command }, ...extra }), encoding: 'utf8', env: { ...process.env, TEMP: directory, TMP: directory, TMPDIR: directory }, windowsHide: true });
      assert.equal(child.status, 0, child.stderr);
      return JSON.parse(child.stdout);
    };
    assert.deepEqual(run('PreToolUse', start), {});
    const shown = run('PostToolUse', 'ccneed kb plan --plan-file input.json', { tool_response: JSON.stringify(plan) });
    assert.match(shown.hookSpecificOutput.additionalContext, /Compare public brand feedback/);
    assert.match(shown.hookSpecificOutput.additionalContext, /Brand/);
    assert.match(shown.hookSpecificOutput.additionalContext, /feedback/);
    assert.deepEqual(run('PreToolUse', start, { permission_mode: 'bypassPermissions' }), {});
    assert.deepEqual(run('PostToolUse', start), {});
    assert.deepEqual(run('PreToolUse', 'rg "ccneed kb start" README.md'), {});
  });
}
