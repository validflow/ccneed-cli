import { readFileSync, mkdirSync, createWriteStream, renameSync, chmodSync, rmSync } from 'node:fs';
import { Readable, Transform } from 'node:stream';
import { pipeline } from 'node:stream/promises';
import { createHash, randomUUID } from 'node:crypto';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { dirname, resolve, join } from 'node:path';
import { EnvHttpProxyAgent } from 'undici';

const versionPattern = /^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-rc\.(0|[1-9][0-9]*))?$/;
const hosts = new Set(['github.com', 'release-assets.githubusercontent.com', 'objects.githubusercontent.com']);
export const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
let downloadDispatcher;

/** assetName 将受支持的平台映射到精确版本的二进制名称。
 * Args:
 *   version: manifest 中的版本，不接受路径或 URL。
 *   platform: Node 操作系统标识。
 *   arch: Node 架构标识。
 * Returns:
 *   固定格式的 Release asset 名称。
 * Errors:
 *   版本或平台不支持时返回异常。
 */
export function assetName(version, platform, arch) {
  if (typeof version !== 'string' || !versionPattern.test(version)) throw new Error('Invalid binary version');
  const os = { linux: 'linux', darwin: 'darwin', win32: 'windows' }[platform];
  const cpu = { x64: 'amd64', arm64: 'arm64' }[arch];
  if (!os || !cpu) throw new Error('Unsupported platform or architecture');
  return 'ccneed-' + version + '-' + os + '-' + cpu + (platform === 'win32' ? '.exe' : '');
}

/** downloadAsset 下载公开 GitHub 产物，限制重定向来源、响应大小和总耗时。
 * Args:
 *   address: 安装器生成的固定 Release 下载地址；代理来自用户既有 HTTP_PROXY、HTTPS_PROXY 和 NO_PROXY 配置。
 * Returns:
 *   共享下载超时的响应流；安装器在写入时限制为 64 MiB。
 * Errors:
 *   非 HTTPS、非允许来源、网络失败、超时或超限返回异常。
 */
export async function downloadAsset(address) {
  const signal = AbortSignal.timeout(60_000);
  let url = new URL(address);
  for (let hop = 0; hop < 4; hop++) {
    if (url.protocol !== 'https:' || url.username || url.password || url.port || !hosts.has(url.hostname)) throw new Error('Untrusted release redirect');
    downloadDispatcher ??= new EnvHttpProxyAgent();
    let response;
    try {
      response = await fetch(url, { redirect: 'manual', signal, dispatcher: downloadDispatcher });
    } catch {
      throw new Error('Release download failed. Check HTTPS_PROXY/HTTP_PROXY settings and access to GitHub.');
    }
    if ([301, 302, 303, 307, 308].includes(response.status)) {
      await response.body?.cancel();
      const location = response.headers.get('location');
      if (!location) throw new Error('Invalid release redirect');
      url = new URL(location, url);
      continue;
    }
    if (!response.ok || !response.body) { await response.body?.cancel(); throw new Error('Release asset unavailable'); }
    return response.body;
  }
  throw new Error('Too many release redirects');
}

/** digest 计算 SHA-256 校验值，不执行待验证文件。
 * Args:
 *   bytes: 二进制内容。
 * Returns:
 *   小写十六进制校验值。
 */
function digest(bytes) { return createHash('sha256').update(bytes).digest('hex'); }

/** installBinary 安装 npm 包自身版本对应的程序，验证成功后才替换本地文件。
 * Args:
 *   options: 包根目录、平台、架构与可注入的下载边界。
 * Returns:
 *   已验证程序的绝对路径。
 * Errors:
 *   缺失校验和、版本不一致、下载或文件操作失败返回异常；不执行未验证程序。
 */
export async function installBinary({ root = packageRoot, platform = process.platform, arch = process.arch, download = downloadAsset } = {}) {
  const manifest = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));
  const asset = assetName(manifest.version, platform, arch);
  const checksums = JSON.parse(readFileSync(join(root, 'checksums.json'), 'utf8'));
  const expected = checksums.assets?.[asset];
  if (checksums.version !== manifest.version || !/^[a-f0-9]{64}$/.test(expected ?? '')) throw new Error('Missing or mismatched release checksum');
  const directory = join(root, 'vendor');
  const target = join(directory, platform === 'win32' ? 'ccneed.exe' : 'ccneed');
  try { if (digest(readFileSync(target)) === expected) return target; } catch (error) { if (error.code !== 'ENOENT') throw error; }
  const content = await download('https://github.com/validflow/ccneed-cli/releases/download/v' + manifest.version + '/' + asset);
  mkdirSync(directory, { recursive: true });
  const temporary = join(directory, '.ccneed-' + randomUUID());
  const hash = createHash('sha256');
  let size = 0;
  const verifier = new Transform({
    /** transform 增量计算校验值，并在超限时终止整条流。
     * Args:
     *   chunk: 当前二进制块。
     *   encoding: Node 提供的编码标记，二进制块不转换编码。
     *   callback: 将验证错误或当前块交给 pipeline。
     * Errors:
     *   超过 64 MiB 时通过 callback 返回错误，不替换既有程序。
     */
    transform(chunk, encoding, callback) {
      size += chunk.byteLength;
      if (size > 64 * 1024 * 1024) { callback(new Error('Release asset exceeds size limit')); return; }
      hash.update(chunk);
      callback(null, chunk);
    },
  });
  try {
    await pipeline(Readable.from(Buffer.isBuffer(content) ? [content] : content), verifier, createWriteStream(temporary, { flags: 'wx', mode: 0o755 }));
    if (hash.digest('hex') !== expected) throw new Error('Release checksum verification failed');
    renameSync(temporary, target);
  } finally { rmSync(temporary, { force: true }); }
  if (platform !== 'win32') chmodSync(target, 0o755);
  return target;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    const manifest = JSON.parse(readFileSync(join(packageRoot, 'package.json'), 'utf8'));
    if (manifest.private) console.error('Development package: binary installation is disabled until release setup is approved.');
    else await installBinary();
  } catch (error) {
    console.error('CCNeed installation failed: ' + error.message);
    process.exitCode = 1;
  }
}
