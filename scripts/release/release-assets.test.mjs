import assert from 'node:assert/strict';
import test from 'node:test';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { createHash } from 'node:crypto';
import { uploadReleaseAsset } from './release-assets.mjs';

test('uploads exact bytes to the release ID returned by GitHub', async () => {
  const out = mkdtempSync(join(tmpdir(), 'ccneed-release-asset-'));
  const file = join(out, 'binary');
  const bytes = Buffer.from('verified bytes');
  writeFileSync(file, bytes);
  const digest = 'sha256:' + createHash('sha256').update(bytes).digest('hex');
  const originalFetch = globalThis.fetch;
  let request;
  globalThis.fetch = async (url, options) => {
    request = { url, options };
    return { status: 201, json: async () => ({ id: 7, name: 'binary', state: 'uploaded', size: bytes.length, digest }) };
  };
  try {
    await uploadReleaseAsset('test-token', 42, 'binary', file);
    assert.equal(request.url, 'https://uploads.github.com/repos/validflow/ccneed-cli/releases/42/assets?name=binary');
    assert.equal(request.options.method, 'POST');
    assert.equal(request.options.redirect, 'error');
    assert.equal(request.options.headers.Authorization, 'Bearer test-token');
    assert.deepEqual(request.options.body, bytes);
  } finally {
    globalThis.fetch = originalFetch;
  }
});

test('rejects a successful HTTP response that describes different asset bytes', async () => {
  const out = mkdtempSync(join(tmpdir(), 'ccneed-release-asset-'));
  const file = join(out, 'binary');
  writeFileSync(file, 'verified bytes');
  const originalFetch = globalThis.fetch;
  globalThis.fetch = async () => ({ status: 201, json: async () => ({ id: 7, name: 'binary', state: 'uploaded', size: 14, digest: 'sha256:' + '0'.repeat(64) }) });
  try {
    await assert.rejects(uploadReleaseAsset('test-token', 42, 'binary', file), /asset differs/);
  } finally {
    globalThis.fetch = originalFetch;
  }
});
