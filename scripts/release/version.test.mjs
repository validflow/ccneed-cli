import assert from 'node:assert/strict';
import test from 'node:test';
import { planRelease, validateManifest, compareVersions } from './version.mjs';

const manifest = (version, extra = {}) => ({ name: '@validflow/ccneed-cli', version, private: false, license: 'MIT', ...extra });

test('only a changed version creates a release plan; no PR title or branch inference', () => {
  assert.deepEqual(planRelease(manifest('0.1.0'), manifest('0.1.0')), { release: false });
  assert.deepEqual(planRelease(manifest('0.1.1'), manifest('0.1.0')), { release: true, version: '0.1.1', tag: 'v0.1.1', channel: 'latest' });
  assert.equal(planRelease(manifest('0.1.0-rc.2'), manifest('0.1.0-rc.1')).channel, 'rc');
});

test('rejects invalid versions, tag mismatch, unpublished placeholder and unapproved license', () => {
  for (const version of ['v0.1.0', '01.2.3', '1.2', '1.2.3+build', '1.2.3-beta.1', '1.2.3-rc.01']) {
    assert.throws(() => validateManifest(manifest(version)), /version/);
  }
  assert.throws(() => validateManifest(manifest('0.0.0')), /placeholder/);
  assert.throws(() => validateManifest(manifest('0.1.0', { private: true })), /private/);
  assert.throws(() => validateManifest(manifest('0.1.0', { license: 'UNLICENSED' })), /license/);
  assert.throws(() => validateManifest(manifest('0.1.0'), 'v0.1.1'), /tag/);
});

test('version ordering rejects downgrade including numeric prerelease identifiers', () => {
  assert.ok(compareVersions('0.1.0-rc.10', '0.1.0-rc.2') > 0);
  assert.ok(compareVersions('0.1.0', '0.1.0-rc.10') > 0);
  assert.ok(compareVersions('0.1.1-rc.1', '0.1.0') > 0);
  assert.throws(() => planRelease(manifest('0.1.0'), manifest('0.2.0')), /increase/);
  assert.throws(() => planRelease(manifest('0.0.0', { private: true }), manifest('0.1.0')), /increase/);
});

test('unchanged placeholder is not a release; enabling first release is explicit', () => {
  assert.deepEqual(planRelease(manifest('0.0.0', { private: true }), null), { release: false });
  assert.equal(planRelease(manifest('0.1.0'), manifest('0.0.0')).release, true);
});
