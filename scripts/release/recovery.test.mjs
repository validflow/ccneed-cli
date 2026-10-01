import assert from 'node:assert/strict';
import test from 'node:test';
import { mkdtempSync, readFileSync, writeFileSync, copyFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { publishGitHub } from './publish.mjs';
const sha = 'a'.repeat(40);
const identity = { version: '0.1.0', tag: 'v0.1.0', channel: 'latest' };

/** boundary 提供真实文件与仅替换外部 gh/API 的发行边界。
 * Args:
 *   release: 模拟的 GitHub 草稿或公开发行状态。
 * Returns:
 *   文件目录、请求记录和外部 I/O 边界。
 */
function boundary(release) {
  const out = mkdtempSync(join(tmpdir(), 'ccneed-release-recovery-'));
  writeFileSync(join(out, 'binary'), 'verified bytes');
  const calls = [];
  const api = async (method, path) => {
    calls.push([method, path]);
    if (path.includes('/git/ref/')) return { status: 200, body: { object: { type: 'commit', sha } } };
    if (method === 'DELETE') {
      const id = Number(path.split('/').at(-1));
      release.assets = release.assets.filter((asset) => asset.id !== id);
      return { status: 204, body: null };
    }
    return { status: 200, body: structuredClone(release) };
  };
  const execute = (binary, args) => {
    assert.equal(binary, 'gh');
    calls.push(args);
    if (args[1] === 'upload') release.assets.push({ name: 'binary', state: 'uploaded', size: 14, id: 9 });
    if (args[1] === 'download') {
      const target = join(args[args.indexOf('--dir') + 1], 'binary');
      copyFileSync(join(out, 'binary'), target);
      if (release.corrupt) writeFileSync(target, 'foreign bytes');
    }
    if (args[1] === 'edit') release.draft = false;
  };
  return { out, calls, api, execute };
}

test('empty starter reservation in draft is safely reconciled, then retries only verify bytes', async () => {
  const release = { tag_name: identity.tag, prerelease: false, draft: true, assets: [{ name: 'binary', state: 'starter', size: 0, id: 7 }] };
  const b = boundary(release);
  for (let attempt = 0; attempt < 3; attempt++) await publishGitHub(identity, sha, b.out, ['binary'], b);
  assert.equal(b.calls.filter((call) => call[0] === 'DELETE').length, 1);
  assert.equal(b.calls.filter((call) => call[1] === 'upload').length, 1);
  assert.equal(b.calls.filter((call) => call[1] === 'edit').length, 1);
  assert.equal(release.draft, false);
  assert.equal(readFileSync(join(b.out, 'binary'), 'utf8'), 'verified bytes');
});

test('public missing/starter assets and completed foreign bytes are never overwritten', async () => {
  for (const release of [
    { draft: false, assets: [] },
    { draft: false, assets: [{ name: 'binary', state: 'starter', size: 0, id: 7 }] },
    { draft: true, corrupt: true, assets: [{ name: 'binary', state: 'uploaded', size: 14, id: 7 }] },
  ]) {
    const b = boundary({ tag_name: identity.tag, prerelease: false, ...release });
    await assert.rejects(publishGitHub(identity, sha, b.out, ['binary'], b));
    assert.equal(b.calls.some((call) => call[0] === 'DELETE' || call[1] === 'upload' || call[1] === 'edit'), false);
  }
});
