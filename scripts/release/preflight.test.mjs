import assert from 'node:assert/strict';
import test from 'node:test';
import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { validateLock, preflight, assertPublicDistribution } from './preflight.mjs';

test('public npm publication requires publicly downloadable GitHub release assets', () => {
  assert.doesNotThrow(() => assertPublicDistribution('false'));
  for (const value of ['true', '', undefined]) assert.throws(() => assertPublicDistribution(value), /public GitHub repository/);
});

test('lock version and package name must match at both root levels', () => {
  const manifest = { name: '@validflow/ccneed-cli', version: '0.1.0' };
  const lock = { ...manifest, packages: { '': { ...manifest } } };
  validateLock(manifest, lock);
  for (const broken of [null, { ...lock, version: '0.2.0' }, { ...lock, packages: { '': { ...manifest, name: '@other/cli' } } }]) assert.throws(() => validateLock(manifest, broken), /lockfile/);
});

test('development scaffold fixture cannot pass a publish preflight', () => {
  const root = mkdtempSync(join(tmpdir(), 'ccneed-preflight-'));
  writeFileSync(join(root, 'package.json'), JSON.stringify({ version: '0.0.0', private: true }));
  assert.throws(() => preflight(root, 'v0.0.0'), /placeholder/);
});

test('ready release requires matching lockfile and each nonempty release material', () => {
  const root = mkdtempSync(join(tmpdir(), 'ccneed-preflight-ready-'));
  const manifest = { name: '@validflow/ccneed-cli', version: '0.1.0', private: false, license: 'MIT' };
  writeFileSync(join(root, 'package.json'), JSON.stringify(manifest));
  writeFileSync(join(root, 'package-lock.json'), JSON.stringify({ ...manifest, packages: { '': manifest } }));
  mkdirSync(join(root, 'skills', 'ccneed'), { recursive: true });
  const files = ['LICENSE', 'CHANGELOG.md', 'skills/ccneed/SKILL.md'];
  for (const file of files) writeFileSync(join(root, file), 'fixture material');
  preflight(root, 'v0.1.0');
  for (const file of files) {
    writeFileSync(join(root, file), '');
    assert.throws(() => preflight(root, 'v0.1.0'), /empty/);
    writeFileSync(join(root, file), 'fixture material');
  }
});
