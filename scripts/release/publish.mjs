import { execFileSync } from 'node:child_process';
import { readFileSync, mkdtempSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { join, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { compareVersions, validateManifest } from './version.mjs';
import { verifyArtifacts, sha256 } from './artifacts.mjs';
import { githubAPI, ensureTag } from './tag.mjs';

/** publicationDecision 判断首次发布或可安全复用的 npm 版本，不接受覆盖和通道回退。
 * Args:
 *   desired: 本地已验证的包名、版本和 SHA-512 integrity。
 *   published: registry 已发布的同版本元数据，或 null。
 *   channel: 目标通道现有元数据，或 null。
 * Returns:
 *   publish 或 reuse。
 * Errors:
 *   内容不一致或通道比待发布版本更新时返回异常。
 */
export function publicationDecision(desired, published, channel) {
  if (channel && compareVersions(channel.version, desired.version) > 0) throw new Error('Refusing npm channel rollback');
  if (!published) return 'publish';
  if (published.name !== desired.name || published.version !== desired.version || published.dist?.integrity !== desired.integrity) throw new Error('Published npm integrity differs; use original artifacts or a new version');
  return 'reuse';
}

/** registry 查询固定官方 registry，不携带产品或 GitHub 凭证。
 * Args:
 *   name: 已校验包名。
 *   selector: 发行版本或稳定/预发布通道。
 * Returns:
 *   公开元数据；明确 404 返回 null。
 * Errors:
 *   网络、重定向、超限或其他状态码失败时返回异常，不能当作不存在。
 */
async function registry(name, selector) {
  const response = await fetch('https://registry.npmjs.org/' + encodeURIComponent(name) + '/' + selector, { redirect: 'error', signal: AbortSignal.timeout(15_000) });
  if (response.status === 404) return null;
  if (!response.ok) throw new Error('npm registry check failed');
  const chunks = [];
  let size = 0;
  for await (const bytes of response.body) {
    size += bytes.length;
    if (size > 1024 * 1024) throw new Error('npm registry response too large');
    chunks.push(bytes);
  }
  const metadata = JSON.parse(Buffer.concat(chunks).toString('utf8'));
  if (metadata.name !== name) throw new Error('npm package identity mismatch');
  return metadata;
}

/** command 执行可信发行工具，使用参数数组而非 shell 拼接。
 * Args:
 *   binary: gh 或 npm。
 *   args: 固定命令与经过校验的发行标识、文件路径。
 * Returns:
 *   标准输出文本。
 * Errors:
 *   命令失败或超时时返回异常。
 */
function command(binary, args) { return execFileSync(binary, args, { encoding: 'utf8', timeout: 120_000, stdio: ['ignore', 'pipe', 'pipe'] }); }

/** publishGitHub 补齐草稿中缺失的同版本产物，校验现有字节后才公开 Release。
 * Args:
 *   identity: manifest 校验后的发行身份。
 *   sha: 已审查 main 提交。
 *   out: 已完成的产物目录。
 *   names: 必须齐全且不可覆盖的文件名。
 *   dependencies: 可注入的远程 API 和外部命令边界。
 * Errors:
 *   已有文件不一致、公开 Release 缺文件或 GH 请求失败时返回异常。
 */
export async function publishGitHub(identity, sha, out, names, { api = githubAPI(process.env.GH_TOKEN), execute = command } = {}) {
  await ensureTag(api, identity.tag, sha);
  let result = await api('GET', '/releases/tags/' + identity.tag, null, [200, 404]);
  if (result.status === 404) {
    const args = ['release', 'create', identity.tag, '--repo', 'validflow/ccneed-cli', '--target', sha, '--verify-tag', '--draft', '--title', identity.tag, '--notes-file', 'CHANGELOG.md'];
    if (identity.channel !== 'latest') args.push('--prerelease');
    execute('gh', args);
    result = await api('GET', '/releases/tags/' + identity.tag);
  }
  const release = result.body;
  if (release.tag_name !== identity.tag || release.prerelease !== (identity.channel !== 'latest')) throw new Error('Existing GitHub release identity mismatch');
  const remote = mkdtempSync(join(out, '.verify-'));
  for (const name of names) {
    const existing = release.assets.find((asset) => asset.name === name);
    if (existing?.state === 'starter') {
      if (!release.draft || existing.size !== 0 || !Number.isSafeInteger(existing.id) || existing.id <= 0) throw new Error('Incomplete release reservation cannot be safely reconciled');
      await api('DELETE', '/releases/assets/' + existing.id, null, [204]);
      execute('gh', ['release', 'upload', identity.tag, join(out, name), '--repo', 'validflow/ccneed-cli']);
    } else if (existing) {
      execute('gh', ['release', 'download', identity.tag, '--repo', 'validflow/ccneed-cli', '--pattern', name, '--dir', remote]);
      if (sha256(join(remote, name)) !== sha256(join(out, name))) throw new Error('Existing release asset differs; never overwrite');
    } else {
      if (!release.draft) throw new Error('Published GitHub release is missing required assets');
      execute('gh', ['release', 'upload', identity.tag, join(out, name), '--repo', 'validflow/ccneed-cli']);
    }
  }
  if (release.draft) execute('gh', ['release', 'edit', identity.tag, '--repo', 'validflow/ccneed-cli', '--draft=false']);
}

/** publish 完成 GitHub 产物与 npm 的顺序发布，重试不移动 tag 或覆盖既有内容。
 * Errors:
 *   任何校验、权限或出版失败返回异常；不谎报完成，不回退 latest。
 */
async function publish() {
  const out = resolve('dist');
  const manifest = JSON.parse(readFileSync(join(out, 'npm', 'package.json'), 'utf8'));
  const identity = validateManifest(manifest, process.env.RELEASE_TAG);
  const sha = process.env.GITHUB_SHA;
  if (!/^[a-f0-9]{40}$/.test(sha)) throw new Error('Missing release revision');
  const tarball = 'ccneed-npm-' + identity.version + '.tgz';
  const bytes = readFileSync(join(out, tarball));
  const desired = { name: manifest.name, version: identity.version, integrity: 'sha512-' + createHash('sha512').update(bytes).digest('base64') };
  const existing = await registry(manifest.name, identity.version);
  publicationDecision(desired, existing, await registry(manifest.name, identity.channel));
  await publishGitHub(identity, sha, out, [...verifyArtifacts(out), 'checksums.json', tarball]);
  // 发布前再次核对通道；workflow 的全局 concurrency 串行化所有版本发行。
  const action = publicationDecision(desired, existing, await registry(manifest.name, identity.channel));
  if (action === 'publish') command('npm', ['publish', join(out, tarball), '--access', 'public', '--tag', identity.channel, '--provenance', '--ignore-scripts']);
  const published = await registry(manifest.name, identity.version);
  if (!published) throw new Error('npm version is not visible yet; retry this exact release run');
  publicationDecision(desired, published, null);
  const current = await registry(manifest.name, identity.channel);
  publicationDecision(desired, published, current);
  if (current?.version !== identity.version) command('npm', ['dist-tag', 'add', manifest.name + '@' + identity.version, identity.channel]);
  if ((await registry(manifest.name, identity.channel))?.version !== identity.version) throw new Error('npm channel update not verified');
  console.log('Verified release ' + identity.tag + ' and npm channel ' + identity.channel);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try { await publish(); } catch (error) { console.error('Release publication failed: ' + error.message); process.exitCode = 1; }
}
