import assert from 'node:assert/strict';
import test from 'node:test';
import { mkdtempSync, readFileSync, statSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { createHash } from 'node:crypto';
import { publishGitHub } from './publish.mjs';
import { sha256 } from './artifacts.mjs';
const sha = 'a'.repeat(40);
const identity = { version: '0.1.0', tag: 'v0.1.0', channel: 'latest' };

/** boundary 提供真实文件与仅替换外部 GitHub API 的发行边界。
 * Args:
 *   release: 模拟的 GitHub 草稿或公开发行状态。
 * Returns:
 *   文件目录、请求记录和外部 I/O 边界。
 */
function boundary(release) {
  const out = mkdtempSync(join(tmpdir(), 'ccneed-release-recovery-'));
  writeFileSync(join(out, 'binary'), 'verified bytes');
  release.id ??= 42;
  release.target_commitish ??= sha;
  const calls = [];
  const api = async (method, path, body) => {
    calls.push([method, path]);
    if (path.includes('/git/ref/')) return { status: 200, body: { object: { type: 'commit', sha } } };
    if (method === 'DELETE') {
      const id = Number(path.split('/').at(-1));
      release.assets = release.assets.filter((asset) => asset.id !== id);
      return { status: 204, body: null };
    }
    if (method === 'PATCH') {
      assert.equal(path, '/releases/' + release.id);
      release.draft = body.draft;
      return { status: 200, body: structuredClone(release) };
    }
    return { status: 200, body: structuredClone(release) };
  };
  const upload = async (id, name, file) => {
    assert.equal(id, release.id);
    calls.push(['UPLOAD', id, name]);
    const asset = { name, state: 'uploaded', size: statSync(file).size, digest: 'sha256:' + sha256(file), id: 9 };
    release.assets.push(asset);
    return asset;
  };
  return { out, calls, api, upload };
}

test('empty starter reservation in draft is safely reconciled, then retries only verify bytes', async () => {
  const release = { tag_name: identity.tag, prerelease: false, draft: true, assets: [{ name: 'binary', state: 'starter', size: 0, id: 7 }] };
  const b = boundary(release);
  for (let attempt = 0; attempt < 3; attempt++) await publishGitHub(identity, sha, b.out, ['binary'], b);
  assert.equal(b.calls.filter((call) => call[0] === 'DELETE').length, 1);
  assert.equal(b.calls.filter((call) => call[0] === 'UPLOAD').length, 1);
  assert.equal(b.calls.filter((call) => call[0] === 'PATCH').length, 1);
  assert.equal(release.draft, false);
  assert.equal(readFileSync(join(b.out, 'binary'), 'utf8'), 'verified bytes');
});

test('public missing/starter assets and completed foreign bytes are never overwritten', async () => {
  for (const release of [
    { draft: false, assets: [] },
    { draft: false, assets: [{ name: 'binary', state: 'starter', size: 0, id: 7 }] },
    { draft: true, assets: [{ name: 'binary', state: 'uploaded', size: 14, digest: 'sha256:' + '0'.repeat(64), id: 7 }] },
  ]) {
    const b = boundary({ tag_name: identity.tag, prerelease: false, ...release });
    await assert.rejects(publishGitHub(identity, sha, b.out, ['binary'], b));
    assert.equal(b.calls.some((call) => ['DELETE', 'UPLOAD', 'PATCH'].includes(call[0])), false);
  }
});

test('existing draft is discovered through release listing when tag lookup returns 404', async () => {
  const release = { tag_name: identity.tag, prerelease: false, draft: true, assets: [] };
  const b = boundary(release);
  const api = async (method, path, body, statuses) => {
    if (path === '/releases/tags/' + identity.tag) {
      b.calls.push([method, path]);
      return release.draft ? { status: 404, body: null } : { status: 200, body: structuredClone(release) };
    }
    if (path.startsWith('/releases?')) {
      b.calls.push([method, path]);
      return { status: 200, body: [structuredClone(release)] };
    }
    return b.api(method, path, body, statuses);
  };
  await publishGitHub(identity, sha, b.out, ['binary'], { ...b, api });
  await publishGitHub(identity, sha, b.out, ['binary'], { ...b, api });

  assert.equal(b.calls.filter((call) => call[0] === 'POST').length, 0);
  assert.equal(b.calls.filter((call) => call[0] === 'UPLOAD').length, 1);
  assert.equal(b.calls.filter((call) => call[0] === 'PATCH').length, 1);
  assert.equal(release.draft, false);
});

test('retry finds an existing draft without creating or replacing its assets', async () => {
  const digest = createHash('sha256').update('verified bytes').digest('hex');
  const release = { tag_name: identity.tag, prerelease: false, draft: true, assets: [{ name: 'binary', state: 'uploaded', size: 14, digest: 'sha256:' + digest, id: 7 }] };
  const b = boundary(release);
  const api = async (method, path, body, statuses) => {
    if (path === '/releases/tags/' + identity.tag) return { status: 404, body: null };
    if (path.startsWith('/releases?')) return { status: 200, body: [structuredClone(release)] };
    return b.api(method, path, body, statuses);
  };

  await publishGitHub(identity, sha, b.out, ['binary'], { ...b, api });

  assert.equal(b.calls.filter((call) => call[0] === 'POST' || call[0] === 'UPLOAD').length, 0);
  assert.equal(b.calls.filter((call) => call[0] === 'PATCH').length, 1);
  assert.equal(release.draft, false);
});

test('new draft publishes from the create response even when both lookups remain empty', async () => {
  const release = { id: 42, tag_name: identity.tag, target_commitish: sha, prerelease: false, draft: true, assets: [] };
  const b = boundary(release);
  const api = async (method, path, body) => {
    b.calls.push([method, path]);
    if (path.includes('/git/ref/')) return { status: 200, body: { object: { type: 'commit', sha } } };
    if (path === '/releases/tags/' + identity.tag) return { status: 404, body: null };
    if (path.startsWith('/releases?')) return { status: 200, body: [] };
    if (method === 'POST' && path === '/releases') {
      assert.equal(body.target_commitish, sha);
      return { status: 201, body: structuredClone(release) };
    }
    if (method === 'PATCH' && path === '/releases/42') {
      release.draft = body.draft;
      return { status: 200, body: structuredClone(release) };
    }
    throw new Error('Unexpected API request: ' + method + ' ' + path);
  };
  const upload = async (id, name, file) => {
    assert.equal(id, 42);
    assert.equal(name, 'binary');
    release.assets.push({ id: 7, name, state: 'uploaded', size: statSync(file).size, digest: 'sha256:' + sha256(file) });
    return release.assets.at(-1);
  };

  await publishGitHub(identity, sha, b.out, ['binary'], { api, upload });

  assert.equal(b.calls.filter((call) => call[0] === 'POST').length, 1);
  assert.equal(b.calls.filter((call) => call[0] === 'PATCH').length, 1);
  assert.equal(b.calls.filter((call) => call[0] === 'GET' && call[1].startsWith('/releases')).length, 2);
  assert.equal(release.draft, false);
});

test('an existing draft from another commit cannot be reused after a tag is recreated', async () => {
  const release = { id: 42, tag_name: identity.tag, target_commitish: 'b'.repeat(40), prerelease: false, draft: true, assets: [] };
  const b = boundary(release);
  await assert.rejects(publishGitHub(identity, sha, b.out, ['binary'], b), /release identity mismatch/i);
  assert.equal(b.calls.some((call) => Array.isArray(call) && ['DELETE', 'PATCH'].includes(call[0])), false);
  assert.equal(b.calls.some((call) => call[0] === 'UPLOAD'), false);
});

test('an old draft is rejected before a replacement tag can be created', async () => {
  const release = { id: 42, tag_name: identity.tag, target_commitish: 'b'.repeat(40), prerelease: false, draft: true, assets: [] };
  const b = boundary(release);
  const api = async (method, path, body) => {
    b.calls.push([method, path]);
    if (path === '/releases/tags/' + identity.tag) return { status: 404, body: null };
    if (path.startsWith('/releases?')) return { status: 200, body: [structuredClone(release)] };
    if (path === '/git/ref/tags/' + identity.tag) return { status: 404, body: null };
    if (method === 'POST' && path === '/git/refs') return { status: 201, body: { object: { type: 'commit', sha } } };
    throw new Error('Unexpected API request: ' + method + ' ' + path);
  };

  await assert.rejects(publishGitHub(identity, sha, b.out, ['binary'], { ...b, api }), /release identity mismatch/i);
  assert.equal(b.calls.some((call) => call[0] === 'POST'), false);
});
