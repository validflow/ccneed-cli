import { readFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
import { validateManifest } from './version.mjs';

/** validateLock 校验锁文件与根包说明书版本和身份同步。
 * Args:
 *   manifest: 根 package.json。
 *   lock: 对应 package-lock.json。
 * Errors:
 *   缺失、身份或版本不一致返回异常。
 */
export function validateLock(manifest, lock) {
  const root = lock?.packages?.[''];
  if (lock?.name !== manifest.name || lock.version !== manifest.version || root?.name !== manifest.name || root.version !== manifest.version) throw new Error('npm lockfile identity/version mismatch');
}

/** assertPublicDistribution 阻止公开 npm 包引用私有仓库的不可下载二进制。
 * Args:
 *   repositoryPrivate: GitHub push 事件提供的仓库私有状态字符串。
 * Errors:
 *   仓库仍私有或事件状态缺失时返回异常，不进入 tag 创建与发布。
 */
export function assertPublicDistribution(repositoryPrivate) {
  if (repositoryPrivate !== 'false') throw new Error('npm publication requires a public GitHub repository');
}

/** preflight 在写 tag 前确认产品 Skill、许可、版本及锁文件已准备好。
 * Args:
 *   root: 可信的仓库根目录。
 *   tag: 待发行 tag。
 * Errors:
 *   任何发行材料不完整时返回异常，禁止创建 tag。
 */
export function preflight(root, tag) {
  const manifest = JSON.parse(readFileSync(root + '/package.json', 'utf8'));
  validateManifest(manifest, tag);
  validateLock(manifest, JSON.parse(readFileSync(root + '/package-lock.json', 'utf8')));
  for (const file of ['LICENSE', 'CHANGELOG.md', 'skills/ccneed/SKILL.md']) {
    if (!readFileSync(root + '/' + file, 'utf8').trim()) throw new Error('Release material is empty: ' + file);
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    assertPublicDistribution(process.env.REPOSITORY_PRIVATE);
    preflight(process.cwd(), process.env.RELEASE_TAG);
  } catch (error) { console.error(error.message); process.exitCode = 1; }
}
