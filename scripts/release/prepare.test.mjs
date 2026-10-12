import assert from 'node:assert/strict';
import test from 'node:test';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { execFileSync } from 'node:child_process';
import { prepare } from './prepare.mjs';

test('real Git revisions distinguish initial scaffold, ordinary commits and release commits', () => {
  const cwd = mkdtempSync(join(tmpdir(), 'ccneed-release-git-'));
  const git = (...args) => execFileSync('git', args, { cwd, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim();
  git('init');
  git('config', 'user.name', 'Release Test');
  git('config', 'user.email', 'test@example.invalid');
  writeFileSync(join(cwd, 'README'), 'test');
  git('add', 'README'); git('commit', '-m', 'init');
  const base = git('rev-parse', 'HEAD');
  const commitManifest = (version, extra = {}) => {
    writeFileSync(join(cwd, 'package.json'), JSON.stringify({ name: '@validflow/ccneed-cli', version, private: false, license: 'MIT', ...extra }));
    git('add', 'package.json'); git('commit', '-m', 'manifest');
    return git('rev-parse', 'HEAD');
  };
  const scaffold = commitManifest('0.0.0', { private: true });
  assert.deepEqual(prepare(base, scaffold, cwd), { release: false });
  const release = commitManifest('0.1.0');
  assert.equal(prepare(scaffold, release, cwd).tag, 'v0.1.0');
  writeFileSync(join(cwd, 'README'), 'ordinary change');
  git('add', 'README'); git('commit', '-m', 'feature');
  const ordinary = git('rev-parse', 'HEAD');
  assert.deepEqual(prepare(release, ordinary, cwd), { release: false });
  assert.throws(() => prepare(release, base, cwd), /checkout/);
  assert.throws(() => prepare('f'.repeat(40), ordinary, cwd));
});
