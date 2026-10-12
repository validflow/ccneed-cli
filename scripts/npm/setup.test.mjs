import assert from 'node:assert/strict';
import test from 'node:test';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, cpSync, symlinkSync, existsSync, lstatSync, readlinkSync } from 'node:fs';
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
  const options = {
    root, home, agents: 'codex,workbuddy,claude',
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
  return { root, home, globalRoot, globalPackage, calls, options };
}

/** writePackage 准备真实的同版本技能载荷，不创建凭证或运行云端任务。
 * Args:
 *   root: 隔离测试包目录。
 *   version: 当前测试版本。
 */
function writePackage(root, version) {
  mkdirSync(join(root, 'skills', 'ccneed', 'references'), { recursive: true });
  writeFileSync(join(root, 'package.json'), JSON.stringify({ name: '@validflow/ccneed-cli', version, private: false, license: 'MIT' }));
  writeFileSync(join(root, 'skills', 'ccneed', 'SKILL.md'), 'skill ' + version);
  for (const name of ['auth', 'account', 'tasks', 'resources']) writeFileSync(join(root, 'skills', 'ccneed', 'references', name + '.md'), name + ' ' + version);
  writeFileSync(join(root, 'checksums.json'), JSON.stringify({ version, skill_sha256: fingerprintSkill(join(root, 'skills', 'ccneed')) }));
}

test('one exact global package supplies all selected user-level hosts; no copy becomes stale on update', async () => {
  const data = fixture();
  const first = await installIntegration(data.options);
  assert.equal(first.version, '0.1.0');
  assert.equal(first.auth, 'not_started');
  assert.equal(first.restart_required, true);
  assert.equal(first.skill_targets.length, 3);
  const targets = [...first.skill_targets];
  for (const target of targets) {
    assert.ok(lstatSync(target).isSymbolicLink());
    assert.equal(readFileSync(join(target, 'references', 'tasks.md'), 'utf8'), 'tasks 0.1.0');
  }
  writePackage(data.root, '0.1.1');
  const second = await installIntegration({ ...data.options, agents: 'codex' });
  assert.equal(second.version, '0.1.1');
  for (const target of targets) {
    assert.equal(readFileSync(join(target, 'SKILL.md'), 'utf8'), 'skill 0.1.1');
    assert.equal(readFileSync(join(target, 'references', 'tasks.md'), 'utf8'), 'tasks 0.1.1');
  }
  assert.equal(data.calls.filter(args => args[0] === 'install').length, 2);
  assert.equal(existsSync(join(data.home, '.ccneed')), false);
});

test('ordinary npm package replacement also updates previously linked hosts without a separate Skill command', async () => {
  const data = fixture();
  const result = await installIntegration(data.options);
  writePackage(data.globalPackage, '0.1.2');
  for (const target of result.skill_targets) assert.equal(readFileSync(join(target, 'SKILL.md'), 'utf8'), 'skill 0.1.2');
});

test('repeating the same version is idempotent; adding a host does not reinstall the global program', async () => {
  const data = fixture();
  const first = await installIntegration({ ...data.options, agents: 'codex' });
  const originalLink = readlinkSync(first.skill_targets[0]);
  await installIntegration(data.options);
  assert.equal(data.calls.filter(args => args[0] === 'install').length, 1);
  assert.equal(readlinkSync(first.skill_targets[0]), originalLink);
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
  const result = await installIntegration(data.options);
  assert.equal(result.version, '0.1.1-rc.1');
  assert.equal(readFileSync(join(result.skill_targets[0], 'SKILL.md'), 'utf8'), 'skill 0.1.1-rc.1');
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
  const first = await installIntegration(data.options);
  writeFileSync(join(first.skill_targets[0], 'references', 'tasks.md'), 'user customization');
  writePackage(data.root, '0.1.1');
  const before = data.calls.filter(args => args[0] === 'install').length;
  await assert.rejects(installIntegration(data.options), /Skill.*checksum|modified Skill/i);
  assert.equal(data.calls.filter(args => args[0] === 'install').length, before);
  assert.equal(readFileSync(join(first.skill_targets[0], 'references', 'tasks.md'), 'utf8'), 'user customization');
});

test('unified installer help works offline without downloading a binary or creating credentials', () => {
  const result = spawnSync(process.execPath, [fileURLToPath(new URL('./run.mjs', import.meta.url)), 'install', '--help'], { encoding: 'utf8', windowsHide: true });
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /--agent/);
  assert.match(result.stdout, /CLI.*Skill/);
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
