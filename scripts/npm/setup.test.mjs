import assert from 'node:assert/strict';
import test from 'node:test';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, cpSync, symlinkSync, existsSync, lstatSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { installIntegration } from './setup.mjs';
import { prepareSkillLinks, attachSkillLinks, parseAgents, fingerprintSkill } from './skills.mjs';

/** fixture 隔离 npm registry/process 边界，保留真实全局包与宿主文件系统链。
 * Args:
 *   version: 合法的本轮发行号。
 * Returns:
 *   源包、模拟全局包、用户目录、调用记录与安装选项。
 */
function fixture(version = '0.1.0') {
  const task = mkdtempSync(join(tmpdir(), 'ccneed-linked-'));
  const root = join(task, 'source');
  const home = join(task, 'user');
  const globalRoot = join(task, 'global');
  const globalPackage = join(globalRoot, '@validflow', 'ccneed-cli');
  mkdirSync(home);
  mkdirSync(globalRoot);
  writePackage(root, version);
  const calls = [];
  const nativeCalls = [];
  let installed = null;
  const options = {
    root, home, agents: 'codex,workbuddy,claude', env: {},
    native: async (host, args) => {
      nativeCalls.push([host, ...args]);
      if (args[1] === 'marketplace') {
        if (args[2] === 'list') return JSON.stringify(host === 'claude' ? [{ name: 'ccneed', installLocation: globalPackage, source: 'directory' }] : { marketplaces: [{ name: 'ccneed', root: globalPackage }] });
        return '{}';
      }
      if (['install', 'update', 'add'].includes(args[1])) {
        installed = { id: 'ccneed@ccneed', pluginId: 'ccneed@ccneed', name: 'ccneed', marketplaceName: 'ccneed', version: JSON.parse(readFileSync(join(globalPackage, 'package.json'))).version, installed: true, enabled: true, scope: 'user', installPath: globalPackage, source: { source: 'local', path: globalPackage } };
        return '{}';
      }
      if (args[1] === 'list') return JSON.stringify(host === 'claude' ? (installed ? [installed] : []) : { installed: installed ? [installed] : [], available: [] });
      throw new Error('unexpected native command: ' + args.join(' '));
    },
    npm: async args => {
      calls.push(args);
      if (args[0] === 'root') return globalRoot + '\n';
      assert.equal(args[2], '@validflow/ccneed-cli@' + JSON.parse(readFileSync(join(root, 'package.json'))).version);
      mkdirSync(join(globalRoot, '@validflow'), { recursive: true });
      cpSync(root, globalPackage, { recursive: true });
      return '';
    },
    verify: async (path, expected) => {
      assert.equal(JSON.parse(readFileSync(join(path, 'package.json'))).version, expected);
      assert.equal(readFileSync(join(path, 'skills', 'ccneed', 'SKILL.md'), 'utf8'), 'skill ' + expected);
    },
  };
  return { root, home, globalRoot, globalPackage, calls, nativeCalls, options };
}

/** writePackage 准备真实的同版本技能载荷，不创建凭证或运行云端任务。
 * Args:
 *   root: 隔离测试包目录。
 *   version: 当前测试版本。
 */
function writePackage(root, version) {
  mkdirSync(join(root, 'skills', 'ccneed', 'scripts'), { recursive: true });
  writeFileSync(join(root, 'skills', 'ccneed', 'scripts', 'acquisition.mjs'), '// Test package hook payload\n');
  mkdirSync(join(root, 'hooks'), { recursive: true });
  writeFileSync(join(root, 'hooks', 'acquisition.mjs'), '// Test package hook payload\n');
  mkdirSync(join(root, 'skills', 'ccneed', 'references'), { recursive: true });
  writeFileSync(join(root, 'package.json'), JSON.stringify({ name: '@validflow/ccneed-cli', version, private: false, license: 'MIT' }));
  for (const [host, directory] of [['claude', '.claude-plugin'], ['codex', '.codex-plugin'], ['workbuddy', '.codebuddy-plugin']]) {
    mkdirSync(join(root, directory), { recursive: true });
    writeFileSync(join(root, directory, 'plugin.json'), JSON.stringify({ name: 'ccneed', version, skills: './skills', hooks: './hooks/' + host + '.json' }));
    writeFileSync(join(root, 'hooks', host + '.json'), JSON.stringify({ hooks: { SessionStart: [], PreToolUse: [], PostToolUse: [] } }));
  }
  for (const directory of ['.claude-plugin', '.codebuddy-plugin', '.agents/plugins']) {
    mkdirSync(join(root, directory), { recursive: true });
    writeFileSync(join(root, directory, 'marketplace.json'), JSON.stringify({ name: 'ccneed', plugins: [{ name: 'ccneed', source: directory === '.agents/plugins' ? { source: 'local', path: './' } : './', version }] }));
  }
  writeFileSync(join(root, 'skills', 'ccneed', 'SKILL.md'), 'skill ' + version);
  for (const name of ['auth', 'account', 'tasks', 'resources']) writeFileSync(join(root, 'skills', 'ccneed', 'references', name + '.md'), name + ' ' + version);
  writeFileSync(join(root, 'checksums.json'), JSON.stringify({ version, skill_sha256: fingerprintSkill(join(root, 'skills', 'ccneed')) }));
}

test('exact global CLI version is coordinated with the selected native plugin manager', async () => {
  const data = fixture();
  const first = await installIntegration({ ...data.options, agents: 'claude' });
  assert.equal(first.version, '0.1.0');
  assert.equal(first.auth, 'not_started');
  assert.equal(first.restart_required, true);
  assert.equal(first.status, 'installed');
  assert.equal(first.plugins[0].host, 'claude');
  assert.equal(first.plugins[0].version, '0.1.0');
  assert.equal(existsSync(join(data.home, '.claude/skills/ccneed')), false);
  writePackage(data.root, '0.1.1');
  const second = await installIntegration({ ...data.options, agents: 'claude' });
  assert.equal(second.version, '0.1.1');
  assert.equal(second.plugins[0].version, '0.1.1');
  assert.ok(data.nativeCalls.some(args => args[0] === 'claude' && args[2] === 'update'));
  assert.equal(data.calls.filter(args => args[0] === 'install').length, 2);
  assert.equal(existsSync(join(data.home, '.ccneed')), false);
});

test('Codex installation checks its marketplace without querying unrelated plugin sources', async () => {
  const data = fixture();
  const native = async (host, args) => {
    if (host === 'codex' && args[0] === 'plugin' && args[1] === 'list'
      && !args.includes('--marketplace')) throw new Error('Unrelated marketplace lookup timed out');
    return data.options.native(host, args);
  };
  const result = await installIntegration({ ...data.options, agents: 'codex', native });
  assert.equal(result.status, 'installed');
  assert.equal(result.plugins[0].version, '0.1.0');
  const lookups = data.nativeCalls.filter(args => args[0] === 'codex' && args[1] === 'plugin' && args[2] === 'list');
  assert.equal(lookups.length, 2, 'installation verifies the plugin both before and after registration');
  for (const args of lookups) assert.equal(args[args.indexOf('--marketplace') + 1], 'ccneed');
});

test('Claude loads Skill and hooks from its native plugin without duplicated user-level entries', async () => {
  const data = fixture();
  mkdirSync(join(data.home, '.claude'), { recursive: true });
  writeFileSync(join(data.home, '.claude', 'settings.json'), JSON.stringify({ permissions: { allow: ['Bash(git status)'] }, env: { KEEP: 'yes' } }));
  const result = await installIntegration({ ...data.options, agents: 'claude' });
  const settings = JSON.parse(readFileSync(join(data.home, '.claude', 'settings.json'), 'utf8'));
  assert.deepEqual(settings.env, { KEEP: 'yes' });
  assert.deepEqual(settings.permissions.allow, ['Bash(git status)']);
  assert.equal(settings.hooks, undefined);
  assert.equal(settings.permissions.ask, undefined);
  assert.equal(result.plugins[0].status, 'installed');
  assert.equal(result.hook_trust_required, true);
});

test('WorkBuddy native desktop action is pending rather than falsely reporting plugin installation', async () => {
  const data = fixture();
  const result = await installIntegration({ ...data.options, agents: 'workbuddy' });
  assert.equal(result.status, 'action_required');
  assert.notEqual(result.plugins[0].status, 'installed');
  assert.equal(result.plugins[0].host, 'workbuddy');
  assert.equal(data.nativeCalls.length, 0, 'unsupported WorkBuddy CLI must never be invented');
  assert.equal(existsSync(join(data.home, '.workbuddy/skills/ccneed')), false);
});

test('native plugin failures are reported per host after the CLI succeeds', async () => {
  const data = fixture();
  const result = await installIntegration({ ...data.options, agents: 'claude,workbuddy', native: async () => { throw new Error('native manager unavailable'); } });
  assert.equal(result.status, 'action_required');
  assert.equal(result.plugins.length, 2);
  assert.ok(result.plugins.every(plugin => plugin.status !== 'installed'));
  assert.equal(JSON.parse(readFileSync(join(data.globalPackage, 'package.json'))).version, result.version);
});

test('Codex verifies actual installed plugin state and keeps native prompt rules', async () => {
  const data = fixture();
  const result = await installIntegration({ ...data.options, agents: 'codex' });
  assert.equal(result.status, 'installed');
  assert.equal(result.plugins[0].host, 'codex');
  const rules = readFileSync(join(data.home, '.codex/rules/ccneed.rules'), 'utf8');
  assert.match(rules, /decision\s*=\s*"prompt"/);
  assert.match(rules, /"kb"\s*,\s*"start"/);
  assert.equal(existsSync(join(data.home, '.codex/hooks.json')), false);
  assert.equal(existsSync(join(data.home, '.agents/skills/ccneed')), false);
});

test('an unavailable, disabled or wrong-version native plugin never counts as complete', async () => {
  for (const variant of ['missing', 'disabled', 'version']) {
    const data = fixture();
    const native = async (host, args) => {
      const output = await data.options.native(host, args);
      if (args[1] !== 'list') return output;
      const plugins = JSON.parse(output);
      if (variant === 'missing') return '[]';
      for (const plugin of plugins) {
        if (variant === 'disabled') plugin.enabled = false;
        if (variant === 'version') plugin.version = '0.0.1';
      }
      return JSON.stringify(plugins);
    };
    const result = await installIntegration({ ...data.options, agents: 'claude', native });
    assert.equal(result.status, 'action_required', variant);
    assert.notEqual(result.plugins[0].status, 'installed', variant);
  }
});

test('a same-name foreign marketplace is preserved and reported as a host failure', async () => {
  const data = fixture();
  const native = async (host, args) => {
    if (args[1] === 'marketplace' && args[2] === 'list') return JSON.stringify([{ name: 'ccneed', source: 'directory', installLocation: join(data.home, 'foreign') }]);
    return data.options.native(host, args);
  };
  const result = await installIntegration({ ...data.options, agents: 'claude', native });
  assert.equal(result.status, 'action_required');
  assert.equal(result.plugins[0].status, 'failed');
  assert.ok(!data.nativeCalls.some(args => args[1] === 'plugin' && ['install', 'update'].includes(args[2])));
});

test('custom native profiles cannot receive a false success with rules written to the default profile', async () => {
  for (const [host, variable, directory] of [['codex', 'CODEX_HOME', '.codex'], ['claude', 'CLAUDE_CONFIG_DIR', '.claude']]) {
    const data = fixture();
    const profile = join(data.home, 'custom-profile');
    const result = await installIntegration({ ...data.options, agents: host, env: { [variable]: profile } });
    assert.equal(result.status, 'action_required');
    assert.equal(result.plugins[0].status, 'action_required');
    assert.equal(data.nativeCalls.length, 0, 'must not register with a mismatched profile');
    assert.equal(existsSync(join(data.home, directory)), false, 'must not write rules to an unused profile');
    assert.equal(existsSync(profile), false);
  }
});

test('a custom profile blocks only that host and an explicit default profile remains supported', async () => {
  const data = fixture();
  const partial = await installIntegration({ ...data.options, agents: 'codex,claude', env: { CODEX_HOME: join(data.home, 'custom') } });
  assert.equal(partial.status, 'action_required');
  assert.equal(partial.plugins.find(plugin => plugin.host === 'claude').status, 'installed');
  assert.ok(data.nativeCalls.every(args => args[0] === 'claude'));
  const complete = await installIntegration({ ...data.options, agents: 'codex', env: { CODEX_HOME: join(data.home, '.codex') } });
  assert.equal(complete.status, 'installed');
});

test('WorkBuddy sandbox uses its native command-safety ask rules without changing isolation', async () => {
  const data = fixture();
  mkdirSync(join(data.home, '.workbuddy'), { recursive: true });
  writeFileSync(join(data.home, '.workbuddy', 'settings.json'), JSON.stringify({ sandbox: { enabled: true, commandSafety: { allow: [{ prefix: ['git', 'status'] }] } } }));
  await installIntegration({ ...data.options, agents: 'workbuddy' });
  const settings = JSON.parse(readFileSync(join(data.home, '.workbuddy', 'settings.json'), 'utf8'));
  assert.equal(settings.sandbox.enabled, true);
  assert.deepEqual(settings.sandbox.commandSafety.allow, [{ prefix: ['git', 'status'] }]);
  assert.ok(settings.sandbox.commandSafety.ask?.some(rule => JSON.stringify(rule.prefix) === JSON.stringify(['ccneed', 'kb', 'start'])));
  assert.ok(settings.sandbox.commandSafety.ask?.some(rule => JSON.stringify(rule.prefix) === JSON.stringify(['ccneed', 'tasks', 'reply'])));
  assert.ok(settings.permissions?.ask?.includes('Bash(ccneed kb start *)'));
  assert.ok(settings.permissions?.ask?.includes('Bash(ccneed tasks reply *)'));
});

test('native migration removes only owned legacy links and handlers after successful registration', async () => {
  const data = fixture();
  cpSync(data.root, data.globalPackage, { recursive: true });
  const [link] = prepareSkillLinks(data.home, ['claude'], data.globalPackage);
  attachSkillLinks([link]);
  const args = [process.execPath, join(data.globalPackage, 'skills/ccneed/scripts/acquisition.mjs'), '--host', 'claude'];
  const posix = args.map(value => "'" + value.replaceAll("'", "'\\''") + "'").join(' ');
  const powershell = '& ' + args.map(value => "'" + value.replaceAll("'", "''") + "'").join(' ');
  const owned = { type: 'command', command: process.platform === 'win32' ? powershell : posix, ...(process.platform === 'win32' ? { shell: 'powershell' } : {}), timeout: 10 };
  const custom = { type: 'command', command: 'echo keep-my-hook' };
  const settings = { env: { KEEP: 'yes' }, hooks: { PreToolUse: [{ matcher: 'Bash|PowerShell|exec_command', hooks: [owned, custom] }] }, permissions: { ask: ['Bash(ccneed kb start *)', 'Bash(git push *)'] } };
  const filename = join(data.home, '.claude/settings.json');
  writeFileSync(filename, JSON.stringify(settings));
  const result = await installIntegration({ ...data.options, agents: 'claude' });
  assert.equal(result.status, 'installed');
  assert.equal(existsSync(link.target), false);
  const actual = JSON.parse(readFileSync(filename, 'utf8'));
  assert.deepEqual(actual.env, settings.env);
  assert.deepEqual(actual.hooks.PreToolUse[0].hooks, [custom]);
  assert.deepEqual(actual.permissions.ask, ['Bash(git push *)']);
});

test('native registration failure preserves the previous working Skill and Hook configuration', async () => {
  const data = fixture();
  cpSync(data.root, data.globalPackage, { recursive: true });
  const [link] = prepareSkillLinks(data.home, ['claude'], data.globalPackage);
  attachSkillLinks([link]);
  const filename = join(data.home, '.claude/settings.json');
  const previous = JSON.stringify({ hooks: { PreToolUse: [{ hooks: [{ type: 'command', command: 'echo previous-hook' }] }] } });
  writeFileSync(filename, previous);
  const result = await installIntegration({ ...data.options, agents: 'claude', native: async () => { throw new Error('native unavailable'); } });
  assert.equal(result.status, 'action_required');
  assert.ok(lstatSync(link.target).isSymbolicLink());
  assert.equal(readFileSync(filename, 'utf8'), previous);
});

test('native manager settings changes survive subsequent permission and migration updates', async () => {
  const data = fixture();
  mkdirSync(join(data.home, '.claude'), { recursive: true });
  const filename = join(data.home, '.claude/settings.json');
  writeFileSync(filename, JSON.stringify({ env: { KEEP: 'yes' } }));
  const native = async (host, args) => {
    const output = await data.options.native(host, args);
    if (args[1] === 'install') writeFileSync(filename, JSON.stringify({ env: { KEEP: 'yes' }, enabledPlugins: { 'ccneed@ccneed': true } }));
    return output;
  };
  const result = await installIntegration({ ...data.options, agents: 'claude', native });
  assert.equal(result.status, 'installed');
  assert.deepEqual(JSON.parse(readFileSync(filename, 'utf8')).enabledPlugins, { 'ccneed@ccneed': true });
});

test('generated Codex hook executes through the platform shell with JSON stdin', () => {
  const directory = mkdtempSync(join(tmpdir(), 'ccneed-native-hook-'));
  const root = fileURLToPath(new URL('../../', import.meta.url));
  const handler = JSON.parse(readFileSync(join(root, 'hooks/codex.json'), 'utf8')).hooks.SessionStart[0].hooks[0];
  const program = process.platform === 'win32' ? 'powershell.exe' : 'sh';
  const args = process.platform === 'win32' ? ['-NoProfile', '-NonInteractive', '-Command', handler.commandWindows] : ['-c', handler.command];
  const result = spawnSync(program, args, { cwd: directory, env: { ...process.env, PLUGIN_ROOT: root }, input: JSON.stringify({ hook_event_name: 'SessionStart', session_id: 'native-protocol-test' }), encoding: 'utf8', timeout: 10000, windowsHide: true });
  assert.equal(result.status, 0, result.stderr);
  assert.equal(JSON.parse(result.stdout).hookSpecificOutput.hookEventName, 'SessionStart');
});

test('only the explicitly selected host receives a native registration command', async () => {
  const data = fixture();
  const result = await installIntegration({ ...data.options, agents: 'claude' });
  assert.equal(result.plugins.length, 1);
  assert.ok(data.nativeCalls.every(args => args[0] === 'claude'));
  assert.equal(existsSync(join(data.home, '.agents')), false);
  assert.equal(existsSync(join(data.home, '.workbuddy')), false);
});

test('repeating the same version is idempotent; adding a host does not reinstall the global program', async () => {
  const data = fixture();
  const first = await installIntegration({ ...data.options, agents: 'claude' });
  const second = await installIntegration({ ...data.options, agents: 'claude' });
  assert.equal(data.calls.filter(args => args[0] === 'install').length, 1);
  assert.equal(second.plugins[0].version, first.plugins[0].version);
  assert.equal(existsSync(join(data.home, '.claude/skills/ccneed')), false);
});

test('unknown hosts, empty selection and malformed arguments never start npm', async () => {
  const data = fixture();
  for (const agents of ['', 'codex,', 'unknown', '__proto__', 'codex --help']) {
    await assert.rejects(installIntegration({ ...data.options, agents }));
    assert.equal(data.calls.length, 0);
  }
  assert.deepEqual(parseAgents('codex,workbuddy,codex'), ['codex', 'workbuddy']);
});

test('custom directories, files and foreign/dangling links are preserved before global installation', async () => {
  for (const kind of ['directory', 'file', 'link']) {
    const data = fixture();
    const parent = join(data.home, '.agents', 'skills');
    mkdirSync(parent, { recursive: true });
    const target = join(parent, 'ccneed');
    if (kind === 'directory') { mkdirSync(target); writeFileSync(join(target, 'SKILL.md'), 'custom'); }
    if (kind === 'file') writeFileSync(target, 'custom');
    if (kind === 'link') symlinkSync(join(data.home, 'missing'), target, process.platform === 'win32' ? 'junction' : 'dir');
    await assert.rejects(installIntegration(data.options), /custom Skill/);
    assert.equal(data.calls.filter(args => args[0] === 'install').length, 0);
    assert.ok(lstatSync(target));
    if (kind === 'directory') assert.equal(readFileSync(join(target, 'SKILL.md'), 'utf8'), 'custom');
  }
});

test('linked parent directories cannot redirect installer writes outside the selected home', async () => {
  const data = fixture();
  const outside = join(data.globalRoot, 'outside');
  mkdirSync(outside);
  symlinkSync(outside, join(data.home, '.agents'), process.platform === 'win32' ? 'junction' : 'dir');
  await assert.rejects(installIntegration(data.options), /plain directory/);
  assert.equal(data.calls.filter(args => args[0] === 'install').length, 0);
  assert.equal(existsSync(join(outside, 'skills')), false);
});

test('missing required bundled references fail before npm; a source link cannot escape package data', async () => {
  const data = fixture();
  writeFileSync(join(data.root, 'skills', 'ccneed', 'references', 'tasks.md'), '');
  await assert.rejects(installIntegration(data.options), /Incomplete/);
  assert.equal(data.calls.length, 0);
  writePackage(data.root, '0.1.0');
  symlinkSync(data.home, join(data.root, 'skills', 'ccneed', 'escape'), process.platform === 'win32' ? 'junction' : 'dir');
  await assert.rejects(installIntegration(data.options), /cannot contain links/);
  assert.equal(data.calls.length, 0);
});

test('npm failures and installed-version/binary mismatches never create host links or report success', async () => {
  for (const failure of ['npm', 'version', 'binary']) {
    const data = fixture();
    const npm = async args => {
      if (args[0] === 'install' && failure === 'npm') throw new Error('npm offline');
      const output = await data.options.npm(args);
      if (args[0] === 'install' && failure === 'version') writePackage(data.globalPackage, '0.2.0');
      return output;
    };
    const verify = async (...args) => { if (failure === 'binary') throw new Error('checksum mismatch'); return data.options.verify(...args); };
    await assert.rejects(installIntegration({ ...data.options, npm, verify }));
    for (const name of ['.agents', '.codebuddy', '.claude']) assert.equal(existsSync(join(data.home, name)), false);
  }
});

test('an older installer cannot downgrade an already installed newer release', async () => {
  const data = fixture('0.1.1');
  writePackage(data.globalPackage, '0.2.0');
  await assert.rejects(installIntegration(data.options), /downgrade/);
  assert.equal(data.calls.filter(args => args[0] === 'install').length, 0);
  assert.equal(JSON.parse(readFileSync(join(data.globalPackage, 'package.json'))).version, '0.2.0');
});

test('next-channel RC installs the exact CLI and Skill version', async () => {
  const data = fixture('0.1.1-rc.1');
  const result = await installIntegration({ ...data.options, agents: 'claude' });
  assert.equal(result.version, '0.1.1-rc.1');
  assert.equal(result.plugins[0].version, '0.1.1-rc.1');
  assert.deepEqual(data.calls.find(args => args[0] === 'install')?.slice(0, 3), ['install', '--global', '@validflow/ccneed-cli@0.1.1-rc.1']);
});

test('RC upgrades follow SemVer order and never replace a newer stable release', async () => {
  const data = fixture('0.1.0-rc.2');
  writePackage(data.globalPackage, '0.1.0');
  await assert.rejects(installIntegration(data.options), /downgrade/);
  assert.equal(data.calls.filter(args => args[0] === 'install').length, 0);
  writePackage(data.root, '0.1.1-rc.1');
  assert.equal((await installIntegration(data.options)).version, '0.1.1-rc.1');
  writePackage(data.root, '0.1.1');
  assert.equal((await installIntegration(data.options)).version, '0.1.1');
});

test('invalid/moving global root and corrupted existing package fail closed', async () => {
  for (const value of ['relative/path', '/', 'C:\\', 'one\ntwo']) {
    const data = fixture();
    await assert.rejects(installIntegration({ ...data.options, npm: async () => value }));
    assert.equal(existsSync(join(data.home, '.agents')), false);
  }
  const data = fixture();
  let reads = 0;
  const npm = async args => args[0] === 'root' && ++reads === 2 ? data.home : data.options.npm(args);
  await assert.rejects(installIntegration({ ...data.options, npm }), /changed/);
  assert.equal(existsSync(join(data.home, '.agents')), false);
  writeFileSync(join(data.globalPackage, 'package.json'), 'broken');
  await assert.rejects(installIntegration(data.options));
});

test('partial host-link failure withdraws only new links; retry completes without replacing existing managed links', async () => {
  const data = fixture();
  cpSync(data.root, data.globalPackage, { recursive: true });
  const plans = prepareSkillLinks(data.home, ['codex', 'workbuddy', 'claude'], data.globalPackage);
  attachSkillLinks([plans[0]]);
  let attempts = 0;
  const failThird = (...args) => { attempts++; if (attempts === 2) throw new Error('denied link'); symlinkSync(...args); };
  assert.throws(() => attachSkillLinks(plans, failThird), /denied link/);
  assert.ok(lstatSync(plans[0].target).isSymbolicLink());
  assert.equal(existsSync(plans[1].target), false);
  assert.equal(existsSync(plans[2].target), false);
  assert.equal(attachSkillLinks(plans).length, 3);
});

test('upgrading preserves edits to the managed previous Skill and stops before npm replaces the package', async () => {
  const data = fixture();
  await installIntegration({ ...data.options, agents: 'claude' });
  const target = join(data.globalPackage, 'skills/ccneed/references/tasks.md');
  writeFileSync(target, 'user customization');
  writePackage(data.root, '0.1.1');
  const before = data.calls.filter(args => args[0] === 'install').length;
  await assert.rejects(installIntegration(data.options), /Skill.*checksum|modified Skill/i);
  assert.equal(data.calls.filter(args => args[0] === 'install').length, before);
  assert.equal(readFileSync(target, 'utf8'), 'user customization');
});

test('unified installer help works offline without downloading a binary or creating credentials', () => {
  const result = spawnSync(process.execPath, [fileURLToPath(new URL('./run.mjs', import.meta.url)), 'install', '--help'], { encoding: 'utf8', windowsHide: true });
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /--agent/);
  assert.match(result.stdout, /CLI.*native plugins/);
});

test('private development package refuses global installation before any npm operation', async () => {
  const data = fixture();
  const manifestPath = join(data.root, 'package.json');
  const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
  manifest.private = true;
  writeFileSync(manifestPath, JSON.stringify(manifest));
  await assert.rejects(installIntegration(data.options), /unpublished|private|Development package/i);
  assert.equal(data.calls.length, 0);
});
