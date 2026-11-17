import assert from 'node:assert/strict';
import test from 'node:test';
import { spawnSync } from 'node:child_process';
import { chmodSync, mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { delimiter, dirname, join } from 'node:path';

/** failedCommand runs the real installer process boundary against an isolated executable.
 * Args:
 *   runner: npm or native runner to exercise, without using a real registry or host profile.
 *   stderr: Fixture child output, including synthetic secrets that must stay private.
 * Returns:
 *   The serialized error exposed by the actual runner.
 */
function failedCommand(runner, stderr) {
  const root = mkdtempSync(join(tmpdir(), 'ccneed-command-'));
  const script = 'process.stderr.write(' + JSON.stringify(stderr) + '); process.exitCode = 1;\n';
  const env = { ...process.env, PATH: root + delimiter + dirname(process.execPath) };
  if (runner === 'npm' && process.platform === 'win32') {
    env.npm_execpath = join(root, 'npm-cli.js');
    writeFileSync(env.npm_execpath, script);
  } else if (runner === 'native' && process.platform === 'win32') {
    const packageRoot = join(root, 'node_modules', '@openai', 'codex');
    mkdirSync(join(packageRoot, 'bin'), { recursive: true });
    writeFileSync(join(root, 'codex.cmd'), 'fixture shim');
    writeFileSync(join(packageRoot, 'package.json'), JSON.stringify({ name: '@openai/codex' }));
    writeFileSync(join(packageRoot, 'bin', 'codex.js'), script);
  } else {
    const executable = join(root, runner === 'npm' ? 'npm' : 'codex');
    writeFileSync(executable, '#!' + process.execPath + '\n' + script);
    chmodSync(executable, 0o700);
  }
  const module = new URL(runner === 'npm' ? './setup.mjs' : './plugin.mjs', import.meta.url).href;
  const name = runner === 'npm' ? 'runNpm' : 'runNative';
  const args = runner === 'npm' ? [['install', '--global', '@validflow/ccneed-cli@0.1.14']] : ['codex', ['plugin', 'marketplace', 'list', '--json']];
  const entry = join(root, 'probe.mjs');
  writeFileSync(entry, 'import { ' + name + ' } from ' + JSON.stringify(module) + ';\ntry { await ' + name + '(...' + JSON.stringify(args) + '); process.exitCode = 2; } catch (error) { console.log(JSON.stringify({ message: error.message, code: error.code, stage: error.stage, exit_code: error.exitCode })); }\n');
  const child = spawnSync(process.execPath, [entry], { env, encoding: 'utf8', timeout: 15_000, windowsHide: true });
  assert.equal(child.status, 0, child.stderr || child.error?.message);
  assert.equal(child.stderr, '', 'raw child diagnostics must not escape');
  return JSON.parse(child.stdout);
}

for (const code of ['EPERM', 'EACCES', 'EBUSY']) {
  for (const prefix of ['npm error', 'npm ERR!']) {
    test('npm numeric exit retains ' + code + ' from ' + prefix + ' without exposing logs', () => {
      const result = failedCommand('npm', prefix + ' code ' + code + '\nAuthorization: Bearer fixture-secret\nnpm error path C:/private/fixture-account/cache\n');
      assert.equal(result.code, code);
      assert.equal(result.stage, 'global npm install');
      assert.equal(result.exit_code, 1);
      assert.match(result.message, new RegExp(code));
      assert.match(result.message, code === 'EBUSY' ? /in use|occupied/i : /permission|write access/i);
      assert.doesNotMatch(JSON.stringify(result), /fixture-secret|fixture-account|Authorization/);
    });
  }
}

test('unknown npm failures retain a bounded exit code and never echo arbitrary output', () => {
  const result = failedCommand('npm', 'npm error code UNKNOWN_FIXTURE\nfixture-secret\n');
  assert.equal(result.code, 'COMMAND_FAILED');
  assert.equal(result.exit_code, 1);
  assert.match(result.message, /npm.*log/i);
  assert.doesNotMatch(JSON.stringify(result), /fixture-secret|UNKNOWN_FIXTURE/);
});

test('native marketplace errors report their stage and safe cause without printing source paths', () => {
  const result = failedCommand('native', 'Error: failed to load marketplace(s):\n- private-marketplace at C:/private/fixture-account\nfixture-secret\n');
  assert.equal(result.stage, 'codex marketplace list');
  assert.equal(result.code, 'MARKETPLACE_UNAVAILABLE');
  assert.match(result.message, /marketplace/i);
  assert.doesNotMatch(JSON.stringify(result), /fixture-secret|fixture-account|private-marketplace/);
});
