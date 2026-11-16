import assert from 'node:assert/strict';
import test from 'node:test';
import { mkdirSync, mkdtempSync, writeFileSync, readFileSync, unlinkSync, existsSync, cpSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { stagePackage, verifyArtifacts, targets } from './artifacts.mjs';
import { publicationDecision, npmPublishArgs } from './publish.mjs';
import { additionalPayloads } from '../npm/additional-plugins.mjs';

const source = fileURLToPath(new URL('../../', import.meta.url));

/** fixture 创建可校验的跨平台产物与同版本 npm 内容，不执行编译或云端写入。
 * Returns:
 *   源目录、产物目录和 manifest。
 */
function fixture() {
  const root = mkdtempSync(join(tmpdir(), 'ccneed-artifacts-'));
  const manifest = { ...JSON.parse(readFileSync(join(source, 'package.json'))), version: '0.1.0' };
  writeFileSync(join(root, 'package.json'), JSON.stringify(manifest));
  for (const path of additionalPayloads) cpSync(join(source, path), join(root, path), { recursive: true });
  const openclawPath = join(root, 'openclaw.plugin.json');
  writeFileSync(openclawPath, JSON.stringify({ ...JSON.parse(readFileSync(openclawPath)), version: manifest.version }));
  for (const host of ['zcode', 'qoder', 'trae']) {
    const path = join(root, host === 'trae' ? 'plugins/trae/.codex-plugin/plugin.json' : '.' + host + '-plugin/plugin.json');
    writeFileSync(path, JSON.stringify({ ...JSON.parse(readFileSync(path)), version: manifest.version }));
  }
  const catalog = JSON.parse(readFileSync(join(root, 'marketplace.json')));
  catalog.plugins.forEach(plugin => { plugin.version = manifest.version; });
  writeFileSync(join(root, 'marketplace.json'), JSON.stringify(catalog));
  for (const name of ['README.md', 'README.zh.md', 'CONTRIBUTING.md', 'SECURITY.md']) writeFileSync(join(root, name), 'fixture');
  for (const language of ['en', 'zh']) {
    mkdirSync(join(root, 'docs', language), { recursive: true });
    writeFileSync(join(root, 'docs', language, 'INSTALL.md'), 'fixture agent installation ' + language);
  }
  mkdirSync(join(root, 'assets'));
  writeFileSync(join(root, 'assets', 'ccneed-logo.png'), Buffer.from('fixture logo'));
  mkdirSync(join(root, 'assets', 'agents'));
  writeFileSync(join(root, 'assets', 'agents', 'kimi.svg'), '<svg/>');
  writeFileSync(join(root, 'LICENSE'), 'fixture license');
  mkdirSync(join(root, 'skills', 'ccneed'), { recursive: true });
  writeFileSync(join(root, 'skills', 'ccneed', 'SKILL.md'), 'fixture skill');
  mkdirSync(join(root, 'skills', 'ccneed', 'references'));
  for (const name of ['auth', 'account', 'tasks', 'resources']) writeFileSync(join(root, 'skills', 'ccneed', 'references', name + '.md'), 'fixture reference');
  mkdirSync(join(root, 'scripts', 'npm'), { recursive: true });
  for (const file of ['install.mjs', 'run.mjs', 'setup.mjs', 'skills.mjs', 'plugin.mjs', 'additional-plugins.mjs']) writeFileSync(join(root, 'scripts', 'npm', file), 'fixture source');
  mkdirSync(join(root, 'hooks'));
  writeFileSync(join(root, 'hooks', 'acquisition.mjs'), 'fixture shared hook');
  mkdirSync(join(root, '.kimi-plugin'));
  writeFileSync(join(root, '.kimi-plugin', 'plugin.json'), JSON.stringify({ name: 'ccneed', version: manifest.version, skills: './skills', sessionStart: { skill: 'ccneed' }, hooks: ['SessionStart', 'PreToolUse', 'PostToolUse'].map(event => ({ event, ...(event === 'SessionStart' ? {} : { matcher: 'Bash|Shell' }), command: 'node ./hooks/acquisition.mjs --host kimi', timeout: 10 })) }));
  for (const [host, directory] of [['claude', '.claude-plugin'], ['codex', '.codex-plugin'], ['workbuddy', '.codebuddy-plugin']]) {
    mkdirSync(join(root, directory));
    writeFileSync(join(root, directory, 'plugin.json'), JSON.stringify({ name: 'ccneed', version: manifest.version, skills: './skills', hooks: './hooks/' + host + '.json' }));
    writeFileSync(join(root, 'hooks', host + '.json'), JSON.stringify({ hooks: { SessionStart: [], PreToolUse: [], PostToolUse: [] } }));
  }
  for (const host of ['zcode', 'qoder', 'trae']) cpSync(join(source, 'hooks', host + '.json'), join(root, 'hooks', host + '.json'));
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
  for (const language of ['en', 'zh']) assert.equal(readFileSync(join(out, 'npm', 'docs', language, 'INSTALL.md'), 'utf8'), 'fixture agent installation ' + language);
  assert.equal(readFileSync(join(out, 'npm', 'README.zh.md'), 'utf8'), 'fixture');
  assert.equal(existsSync(join(out, 'npm', 'INSTALL.md')), false);
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
  assert.equal(JSON.parse(readFileSync(join(staged, '.kimi-plugin/plugin.json'), 'utf8')).version, manifest.version);
  assert.equal(existsSync(join(staged, 'kimi.plugin.json')), false);
  assert.equal(readFileSync(join(staged, 'assets/agents/kimi.svg'), 'utf8'), '<svg/>');
  for (const path of additionalPayloads) assert.equal(existsSync(join(staged, path)), true, path);
  assert.equal(readFileSync(join(staged, 'scripts/npm/additional-plugins.mjs'), 'utf8'), 'fixture source');
});

test('release rejects an incomplete native package rather than shipping only a Skill', () => {
  for (const entry of ['plugins/pi.mjs', 'plugins/openclaw.mjs', 'plugins/dsh/index.mjs', 'hooks/qoder.json']) {
    const { root, out, manifest } = fixture();
    unlinkSync(join(root, entry));
    assert.throws(() => stagePackage(root, out, manifest), /ENOENT|plugin/i, entry);
  }
});

test('Kimi version or missing collection guard blocks publication', () => {
  for (const change of [{ version: '0.0.1' }, { hooks: [] }]) {
    const { root, out, manifest } = fixture();
    const path = join(root, '.kimi-plugin/plugin.json');
    writeFileSync(path, JSON.stringify({ ...JSON.parse(readFileSync(path)), ...change }));
    assert.throws(() => stagePackage(root, out, manifest), /Kimi/);
  }
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
