import assert from 'node:assert/strict';
import test from 'node:test';
import { cpSync, existsSync, mkdtempSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { installIntegration } from './setup.mjs';
import { fingerprintSkill } from './skills.mjs';
import { stageTraePayload } from './additional-plugins.mjs';

const source = fileURLToPath(new URL('../../', import.meta.url));

test('new native hosts never enter legacy profile migration or claim unverified installation', async () => {
  const task = mkdtempSync(join(tmpdir(), 'ccneed-additional-install-'));
  const root = join(task, 'source');
  const home = join(task, 'user');
  const globalRoot = join(task, 'global');
  const globalPackage = join(globalRoot, '@validflow', 'ccneed-cli');
  mkdirSync(home);
  mkdirSync(root);
  for (const entry of ['package.json', 'skills', 'hooks', 'plugins', 'openclaw.plugin.json', 'marketplace.json', '.zcode-plugin', '.qoder-plugin']) {
    if (existsSync(join(source, entry))) cpSync(join(source, entry), join(root, entry), { recursive: true });
  }
  const version = JSON.parse(readFileSync(join(root, 'package.json'))).version;
  stageTraePayload(root, root);
  writeFileSync(join(root, 'checksums.json'), JSON.stringify({ version, skill_sha256: fingerprintSkill(join(root, 'skills/ccneed')) }));
  const result = await installIntegration({ root, home, agents: 'openclaw,pi,zcode,qoder,trae,dsh', env: {},
    npm: async args => {
      if (args[0] === 'root') return globalRoot;
      assert.deepEqual(args.slice(0, 3), ['install', '--global', '@validflow/ccneed-cli@' + version]);
      mkdirSync(join(globalRoot, '@validflow'), { recursive: true });
      cpSync(root, globalPackage, { recursive: true });
      return '';
    },
    verify: async path => assert.equal(path, globalPackage),
    native: async () => { throw new Error('No host process installed in isolated test'); },
  });
  assert.equal(result.status, 'action_required');
  assert.equal(result.plugins.length, 6);
  for (const plugin of result.plugins) {
    assert.equal(plugin.status, 'action_required', plugin.host);
    assert.equal(plugin.requested_version, version);
    assert.ok(plugin.next_steps.some(step => step.includes(globalPackage.replaceAll('\\', '/')) || step.includes(globalPackage)), plugin.host);
    assert.notEqual(plugin.error, 'legacy_integration_cleanup_required');
    assert.equal(plugin.version, null);
  }
  for (const directory of ['.openclaw', '.pi', '.zcode', '.qoder', '.trae', '.dsh', '.codex', '.claude', '.workbuddy']) assert.equal(existsSync(join(home, directory)), false);
});
