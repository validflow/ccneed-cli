import assert from 'node:assert/strict';
import test from 'node:test';
import { notifyMultica, releaseNotification } from './notify.mjs';

const manifest = { name: '@validflow/ccneed-cli', version: '0.1.2', private: false, license: 'MIT' };
const env = {
  RELEASE_TAG: 'v0.1.2',
  GITHUB_REPOSITORY: 'validflow/ccneed-cli',
  GITHUB_SHA: 'a'.repeat(40),
  GITHUB_RUN_ID: '12345',
  MULTICA_CLI_RELEASE_WEBHOOK_URL: 'https://multica.ai/api/webhooks/autopilots/test-token',
};

test('stable publication sends a bounded event with a version idempotency key', async () => {
  let requests = 0;
  const result = await notifyMultica(env, async (url, options) => {
    requests++;
    assert.equal(url, env.MULTICA_CLI_RELEASE_WEBHOOK_URL);
    assert.equal(options.headers['Idempotency-Key'], 'ccneed-cli:v0.1.2');
    assert.equal(options.redirect, 'error');
    assert.equal(JSON.parse(options.body).event, 'ccneed-cli.release.verified');
    assert.equal(JSON.parse(options.body).version, '0.1.2');
    return { ok: true, json: async () => ({ status: 'accepted' }) };
  }, manifest);
  assert.equal(result, 'accepted');
  assert.equal(requests, 1);
  assert.equal(releaseNotification(manifest, env).payload.channel, 'latest');
});

test('RC releases do not notify the production group', async () => {
  const result = await notifyMultica({ ...env, RELEASE_TAG: 'v0.1.3-rc.1' }, () => { throw new Error('unexpected request'); }, { ...manifest, version: '0.1.3-rc.1' });
  assert.equal(result, 'skipped');
});

test('rejects missing credentials and ignored deliveries', async () => {
  await assert.rejects(notifyMultica({ ...env, MULTICA_CLI_RELEASE_WEBHOOK_URL: '' }, () => {}, manifest), /webhook secret/);
  await assert.rejects(notifyMultica({ ...env, MULTICA_CLI_RELEASE_WEBHOOK_URL: 'https://example.com/hook' }, () => {}, manifest), /destination/);
  await assert.rejects(notifyMultica(env, async () => ({ ok: true, json: async () => ({ status: 'ignored' }) }), manifest), /did not start/);
  assert.equal(await notifyMultica(env, async () => ({ ok: true, json: async () => ({ status: 'duplicate' }) }), manifest), 'duplicate');
});

test('rejects release identity drift', () => {
  assert.throws(() => releaseNotification(manifest, { ...env, RELEASE_TAG: 'v0.1.1' }), /differs/);
  assert.throws(() => releaseNotification(manifest, { ...env, GITHUB_REPOSITORY: 'other/repo' }), /identity/);
});
