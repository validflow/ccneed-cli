import assert from 'node:assert/strict';
import test from 'node:test';
import { mkdirSync, mkdtempSync, writeFileSync, readFileSync, unlinkSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { stagePackage, verifyArtifacts, targets } from './artifacts.mjs';
import { publicationDecision, npmPublishArgs } from './publish.mjs';

/** fixture 创建可校验的跨平台产物与同版本 npm 内容，不执行编译或云端写入。
 * Returns:
 *   源目录、产物目录和 manifest。
 */
function fixture() {
  const root = mkdtempSync(join(tmpdir(), 'ccneed-artifacts-'));
  const manifest = { name: '@validflow/ccneed-cli', version: '0.1.0', private: false, license: 'MIT' };
  writeFileSync(join(root, 'package.json'), JSON.stringify(manifest));
  writeFileSync(join(root, 'README.md'), 'fixture');
  writeFileSync(join(root, 'INSTALL.md'), 'fixture agent installation');
  mkdirSync(join(root, 'assets'));
  writeFileSync(join(root, 'assets', 'ccneed-logo.png'), Buffer.from('fixture logo'));
  writeFileSync(join(root, 'LICENSE'), 'fixture license');
  mkdirSync(join(root, 'skills', 'ccneed'), { recursive: true });
  writeFileSync(join(root, 'skills', 'ccneed', 'SKILL.md'), 'fixture skill');
  mkdirSync(join(root, 'skills', 'ccneed', 'references'));
  for (const name of ['auth', 'account', 'tasks', 'resources']) writeFileSync(join(root, 'skills', 'ccneed', 'references', name + '.md'), 'fixture reference');
  mkdirSync(join(root, 'scripts', 'npm'), { recursive: true });
  for (const file of ['install.mjs', 'run.mjs', 'setup.mjs', 'skills.mjs', 'plugin.mjs']) writeFileSync(join(root, 'scripts', 'npm', file), 'fixture source');
  mkdirSync(join(root, 'hooks'));
  writeFileSync(join(root, 'hooks', 'acquisition.mjs'), 'fixture shared hook');
  for (const [host, directory] of [['claude', '.claude-plugin'], ['codex', '.codex-plugin'], ['workbuddy', '.codebuddy-plugin']]) {
    mkdirSync(join(root, directory));
    writeFileSync(join(root, directory, 'plugin.json'), JSON.stringify({ name: 'ccneed', version: manifest.version, skills: './skills', hooks: './hooks/' + host + '.json' }));
    writeFileSync(join(root, 'hooks', host + '.json'), JSON.stringify({ hooks: { SessionStart: [], PreToolUse: [], PostToolUse: [] } }));
  }
  for (const directory of ['.claude-plugin', '.codebuddy-plugin', '.agents/plugins']) {
    mkdirSync(join(root, directory), { recursive: true });
    writeFileSync(join(root, directory, 'marketplace.json'), JSON.stringify({ name: 'ccneed', plugins: [{ name: 'ccneed', source: './', version: manifest.version }] }));
  }
  const out = join(root, 'dist');
  mkdirSync(out);
  for (const target of targets('0.1.0')) writeFileSync(join(out, target.asset), Buffer.from(target.asset));
  return { root, out, manifest };
}

test('npm staging includes same-source Skill and complete checksum map for all six targets', () => {
  const { root, out, manifest } = fixture();
  stagePackage(root, out, manifest);
  assert.equal(verifyArtifacts(out).length, 6);
  for (const file of ['setup.mjs', 'skills.mjs']) assert.equal(readFileSync(join(out, 'npm', 'scripts', 'npm', file), 'utf8'), 'fixture source');
  assert.equal(readFileSync(join(out, 'npm', 'skills', 'ccneed', 'SKILL.md'), 'utf8'), 'fixture skill');
  assert.equal(readFileSync(join(out, 'npm', 'INSTALL.md'), 'utf8'), 'fixture agent installation');
  assert.deepEqual(readFileSync(join(out, 'npm', 'assets', 'ccneed-logo.png')), Buffer.from('fixture logo'));
  assert.equal(JSON.parse(readFileSync(join(out, 'npm', 'package.json'))).version, '0.1.0');
  writeFileSync(join(out, targets('0.1.0')[0].asset), 'corrupt');
  assert.throws(() => verifyArtifacts(out), /checksum/);
});

test('staging fails when matching Skill is unavailable; never publishes CLI alone', () => {
  const { root, out, manifest } = fixture();
  writeFileSync(join(root, 'skills', 'ccneed', 'SKILL.md'), '');
  assert.throws(() => stagePackage(root, out, manifest), /usable Skill/);
  unlinkSync(join(root, 'skills', 'ccneed', 'SKILL.md'));
  assert.throws(() => stagePackage(root, out, manifest), /ENOENT/);
  assert.throws(() => stagePackage(root, out, { ...manifest, version: '0.2.0' }), /version mismatch/);
});

test('release staging includes installer imports and all native plugin payloads', () => {
  const { root, out, manifest } = fixture();
  const staged = stagePackage(root, out, manifest);
  assert.equal(readFileSync(join(staged, 'scripts/npm/plugin.mjs'), 'utf8'), 'fixture source');
  assert.equal(readFileSync(join(staged, 'hooks/acquisition.mjs'), 'utf8'), 'fixture shared hook');
  for (const directory of ['.claude-plugin', '.codex-plugin', '.codebuddy-plugin']) {
    assert.equal(JSON.parse(readFileSync(join(staged, directory, 'plugin.json'), 'utf8')).version, manifest.version);
  }
  assert.equal(JSON.parse(readFileSync(join(staged, '.agents/plugins/marketplace.json'), 'utf8')).name, 'ccneed');
});

test('mismatched native plugin versions and missing hook payloads block publication', () => {
  const { root, out, manifest } = fixture();
  writeFileSync(join(root, '.codex-plugin/plugin.json'), JSON.stringify({ name: 'ccneed', version: '0.0.1', skills: './skills', hooks: './hooks/codex.json' }));
  assert.throws(() => stagePackage(root, out, manifest), /plugin.*version/i);
  writeFileSync(join(root, '.codex-plugin/plugin.json'), JSON.stringify({ name: 'ccneed', version: manifest.version, skills: './skills', hooks: './hooks/codex.json' }));
  unlinkSync(join(root, 'hooks/acquisition.mjs'));
  assert.throws(() => stagePackage(root, out, manifest));
});

test('npm retry is idempotent only when published version and exact tarball match', () => {
  const desired = { name: '@validflow/ccneed-cli', version: '0.1.1', integrity: 'sha512-fixture' };
  assert.equal(publicationDecision(desired, null, { version: '0.1.0' }), 'publish');
  assert.equal(publicationDecision(desired, { name: desired.name, version: desired.version, dist: { integrity: desired.integrity } }, { version: desired.version }), 'reuse');
  assert.throws(() => publicationDecision(desired, { name: desired.name, version: desired.version, dist: { integrity: 'sha512-other' } }, null), /integrity/);
  assert.throws(() => publicationDecision(desired, null, { version: '0.2.0' }), /rollback/);
});

test('npm publication explicitly selects stable or prerelease channel without private-repo provenance', () => {
  assert.deepEqual(npmPublishArgs('/tmp/ccneed.tgz', 'next'), ['publish', '/tmp/ccneed.tgz', '--access', 'public', '--tag', 'next', '--ignore-scripts']);
  assert.deepEqual(npmPublishArgs('/tmp/ccneed.tgz', 'latest'), ['publish', '/tmp/ccneed.tgz', '--access', 'public', '--tag', 'latest', '--ignore-scripts']);
  assert.throws(() => npmPublishArgs('/tmp/ccneed.tgz', 'rc'), /channel/);
});

test('a release cannot stage a Skill whose required task reference is missing', () => {
  const { root, out, manifest } = fixture();
  unlinkSync(join(root, 'skills', 'ccneed', 'references', 'tasks.md'));
  assert.throws(() => stagePackage(root, out, manifest));
});
