import assert from 'node:assert/strict';
import test from 'node:test';
import { createHash } from 'node:crypto';
import { mkdtempSync, readFileSync, writeFileSync, mkdirSync, existsSync, readdirSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { createServer } from 'node:http';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { assetName, installBinary, downloadAsset } from './install.mjs';
const bytes = Buffer.from('verified fixture executable');
const hash = createHash('sha256').update(bytes).digest('hex');

for (const variable of ['HTTPS_PROXY', 'HTTP_PROXY']) {
  test(`release downloader uses ${variable} for HTTPS without requiring a newer Node runtime`, { timeout: 10_000 }, async (t) => {
    const authorities = [];
    const proxy = createServer();
    proxy.on('connect', (request, socket) => {
      authorities.push(request.url);
      socket.end('HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\nConnection: close\r\n\r\n');
    });
    await new Promise(resolve => proxy.listen(0, '127.0.0.1', resolve));
    t.after(() => new Promise(resolve => proxy.close(resolve)));
    const environment = Object.fromEntries(Object.entries(process.env).filter(([name]) => !/^(https?_proxy|all_proxy|no_proxy)$/i.test(name)));
    environment[variable] = `http://127.0.0.1:${proxy.address().port}`;
    environment.NO_PROXY = '';
    const script = `import { downloadAsset } from ${JSON.stringify(new URL('./install.mjs', import.meta.url).href)};
      try { await downloadAsset('https://github.com/validflow/ccneed-cli/releases/download/v0.1.5/not-a-real-asset'); process.exitCode = 2; }
      catch { console.log('download rejected'); }`;
    let result;
    try {
      result = await promisify(execFile)(process.execPath, ['--input-type=module', '-e', script], { env: environment, timeout: 4_000, windowsHide: true });
    } catch (error) { result = error; }
    assert.ok(authorities.length > 0, `${variable} must route the HTTPS request through the configured proxy`);
    assert.ok(authorities.every(authority => authority === 'github.com:443'));
    assert.equal(result.stdout.trim(), 'download rejected');
  });
}

test('real downloader follows only trusted HTTPS redirects and returns a stream', async (t) => {
  const requests = [];
  t.mock.method(globalThis, 'fetch', async (url, options) => {
    requests.push(String(url));
    assert.equal(options.redirect, 'manual');
    assert.ok(options.signal instanceof AbortSignal);
    return requests.length === 1 ? new Response(null, { status: 302, headers: { location: 'https://release-assets.githubusercontent.com/fixture' } }) : new Response(bytes);
  });
  const body = await downloadAsset('https://github.com/validflow/ccneed-cli/releases/download/v0.1.1/binary');
  assert.ok(body instanceof ReadableStream);
  const chunks = [];
  for await (const chunk of body) chunks.push(chunk);
  assert.deepEqual(Buffer.concat(chunks), bytes);
  assert.equal(requests.length, 2);
});

test('foreign, plaintext and credentialed redirects stop before following them', async (t) => {
  for (const location of ['https://evil.invalid/file', 'http://github.com/file', 'https://user:password@github.com/file', 'https://github.com:444/file']) {
    let calls = 0;
    t.mock.method(globalThis, 'fetch', async () => { calls++; return new Response(null, { status: 302, headers: { location } }); });
    await assert.rejects(downloadAsset('https://github.com/fixture'), /Untrusted/);
    assert.equal(calls, 1);
    t.mock.restoreAll();
  }
});

test('download failures explain proxy configuration without exposing the original network error', async (t) => {
  const privateDetail = 'http://private-user:private-password@proxy.invalid';
  t.mock.method(globalThis, 'fetch', async () => { throw new Error(privateDetail); });
  await assert.rejects(downloadAsset('https://github.com/fixture'), error => {
    assert.match(error.message, /HTTPS_PROXY\/HTTP_PROXY/);
    assert.match(error.message, /GitHub/);
    assert.equal(error.message.includes(privateDetail), false);
    return true;
  });
});

test('stream errors remove temporary data and preserve the previous executable', async () => {
  const root = fixture();
  mkdirSync(join(root, 'vendor'));
  const target = join(root, 'vendor', 'ccneed');
  writeFileSync(target, 'previous executable');
  async function* failAfterChunk() { yield bytes; throw new Error('interrupted stream'); }
  await assert.rejects(installBinary({ root, platform: 'linux', arch: 'x64', download: async () => failAfterChunk() }), /interrupted/);
  assert.equal(readFileSync(target, 'utf8'), 'previous executable');
  assert.deepEqual(readdirSync(join(root, 'vendor')), ['ccneed']);
});

test('oversized streamed downloads are rejected and leave no partial executable', async () => {
  const root = fixture();
  async function* oversized() { const chunk = Buffer.alloc(1024 * 1024); for (let i = 0; i < 65; i++) yield chunk; }
  await assert.rejects(installBinary({ root, platform: 'linux', arch: 'x64', download: async () => oversized() }), /size limit/);
  assert.deepEqual(readdirSync(join(root, 'vendor')), []);
});

test('invalid redirects and redirect loops fail within the request bound', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => new Response(null, { status: 302 }));
  await assert.rejects(downloadAsset('https://github.com/fixture'), /Invalid release redirect/);
  t.mock.restoreAll();
  let calls = 0;
  t.mock.method(globalThis, 'fetch', async () => { calls++; return new Response(null, { status: 302, headers: { location: '/fixture' } }); });
  await assert.rejects(downloadAsset('https://github.com/fixture'), /Too many/);
  assert.equal(calls, 4);
});

/** fixture 创建无外部服务或产品凭证的本地 npm 包边界。
 * Returns:
 *   包目录与安装目标。
 */
function fixture() {
  const root = mkdtempSync(join(tmpdir(), 'ccneed-install-'));
  writeFileSync(join(root, 'package.json'), JSON.stringify({ version: '0.1.1' }));
  writeFileSync(join(root, 'checksums.json'), JSON.stringify({ version: '0.1.1', assets: { 'ccneed-0.1.1-linux-amd64': hash } }));
  return root;
}

test('asset mapping is explicit, unsupported targets fail safely', () => {
  assert.equal(assetName('0.1.1', 'win32', 'arm64'), 'ccneed-0.1.1-windows-arm64.exe');
  assert.equal(assetName('0.1.1', 'darwin', 'x64'), 'ccneed-0.1.1-darwin-amd64');
  assert.throws(() => assetName('0.1.1', 'linux', 'ia32'), /Unsupported/);
  assert.throws(() => assetName('../escape', 'linux', 'x64'), /version/);
});

test('installer downloads exact manifest version, verifies and reuses trusted local bytes', async () => {
  const root = fixture();
  let calls = 0;
  const download = async (url) => {
    calls++;
    assert.equal(url, 'https://github.com/validflow/ccneed-cli/releases/download/v0.1.1/ccneed-0.1.1-linux-amd64');
    return bytes;
  };
  const target = await installBinary({ root, platform: 'linux', arch: 'x64', download });
  assert.deepEqual(readFileSync(target), bytes);
  await installBinary({ root, platform: 'linux', arch: 'x64', download });
  assert.equal(calls, 1);
  writeFileSync(target, 'tampered');
  await installBinary({ root, platform: 'linux', arch: 'x64', download });
  assert.equal(calls, 2);
});

test('missing/wrong checksums and network failure never install or replace a binary', async () => {
  for (const download of [async () => Buffer.from('wrong'), async () => { throw new Error('offline'); }]) {
    const root = fixture();
    mkdirSync(join(root, 'vendor'));
    const target = join(root, 'vendor', 'ccneed');
    writeFileSync(target, 'existing');
    await assert.rejects(installBinary({ root, platform: 'linux', arch: 'x64', download }));
    assert.equal(readFileSync(target, 'utf8'), 'existing');
  }
  const root = fixture();
  writeFileSync(join(root, 'checksums.json'), JSON.stringify({ version: '0.1.0', assets: {} }));
  await assert.rejects(installBinary({ root, platform: 'linux', arch: 'x64', download: async () => { assert.fail('must not download'); } }), /checksum/);
  assert.equal(existsSync(join(root, 'vendor', 'ccneed')), false);
});
