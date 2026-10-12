import { readFileSync, statSync } from 'node:fs';
import { sha256 } from './artifacts.mjs';

/** assertAssetMatches 校验已上传资产与本次构建完全一致，禁止覆盖不同字节。
 * Args:
 *   asset: GitHub Release 返回的资产元数据。
 *   name: 已验证的发行文件名。
 *   file: 本次构建的本地文件路径。
 * Errors:
 *   资产状态、大小或 SHA-256 摘要不一致时返回异常。
 */
export function assertAssetMatches(asset, name, file) {
  if (asset?.name !== name || asset.state !== 'uploaded' || asset.size !== statSync(file).size || asset.digest !== 'sha256:' + sha256(file)) {
    throw new Error('Existing release asset differs; never overwrite');
  }
}

/** uploadReleaseAsset 按创建 Release 返回的 ID 上传资产，不通过 tag 再查询草稿。
 * Args:
 *   token: 当前 Actions 的短期 GitHub token。
 *   releaseId: GitHub 创建 Release 响应中的数字 ID。
 *   name: 已验证的发行文件名。
 *   file: 本次构建的本地文件路径。
 * Returns:
 *   与本地文件摘要一致的已上传资产元数据。
 * Errors:
 *   身份、上传状态或返回资产内容不一致时返回异常。
 */
export async function uploadReleaseAsset(token, releaseId, name, file) {
  if (!token || !Number.isSafeInteger(releaseId) || releaseId <= 0) throw new Error('Invalid release asset request');
  const bytes = readFileSync(file);
  const url = 'https://uploads.github.com/repos/validflow/ccneed-cli/releases/' + releaseId + '/assets?name=' + encodeURIComponent(name);
  const response = await fetch(url, {
    method: 'POST',
    redirect: 'error',
    signal: AbortSignal.timeout(120_000),
    headers: {
      Authorization: 'Bearer ' + token,
      Accept: 'application/vnd.github+json',
      'Content-Type': 'application/octet-stream',
    },
    body: bytes,
  });
  if (response.status !== 201) throw new Error('Release asset upload failed with status ' + response.status);
  const asset = await response.json();
  if (!Number.isSafeInteger(asset.id) || asset.id <= 0) throw new Error('Invalid uploaded release asset');
  assertAssetMatches(asset, name, file);
  return asset;
}
