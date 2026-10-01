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
  try { preflight(process.cwd(), process.env.RELEASE_TAG); } catch (error) { console.error(error.message); process.exitCode = 1; }
}
