import assert from 'node:assert/strict';
import test from 'node:test';
import { ensureTag } from './tag.mjs';
const sha = 'a'.repeat(40);
const ref = (value = sha) => ({ object: { type: 'commit', sha: value } });

test('create once then reuse tag on retry, without moving it', async () => {
  let stored;
  let writes = 0;
  const api = async (method, path, body) => {
    if (method === 'POST') {
      assert.equal(path, '/git/refs');
      assert.deepEqual(body, { ref: 'refs/tags/v0.1.0', sha });
      writes++;
      stored = ref();
      return { status: 201, body: stored };
    }
    assert.equal(path, '/git/ref/tags/v0.1.0');
    return { status: stored ? 200 : 404, body: stored ?? {} };
  };
  assert.equal(await ensureTag(api, 'v0.1.0', sha), 'created');
  assert.equal(await ensureTag(api, 'v0.1.0', sha), 'reused');
  assert.equal(writes, 1);
});

test('reject foreign or annotated tag without writes', async () => {
  for (const existing of [ref('b'.repeat(40)), { object: { type: 'tag', sha } }]) {
    await assert.rejects(ensureTag(async (method) => {
      assert.equal(method, 'GET');
      return { status: 200, body: existing };
    }, 'v0.1.0', sha), /another commit/);
  }
});

test('creation race revalidates exact revision', async () => {
  const results = [{ status: 404, body: {} }, { status: 422, body: {} }, { status: 200, body: ref() }];
  assert.equal(await ensureTag(async () => results.shift(), 'v0.1.0', sha), 'reused');
  assert.equal(results.length, 0);
});
