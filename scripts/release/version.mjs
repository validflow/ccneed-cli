const pattern = /^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-rc\.(0|[1-9][0-9]*))?$/;

/** parseVersion 校验本仓发行格式，拒绝路径字符和非约定的版本后缀。
 * Args:
 *   value: 无 v 前缀的稳定版本或 rc 版本。
 * Returns:
 *   数字版本分量与可选 rc 序号。
 * Errors:
 *   非法版本返回异常，不猜测或修正输入。
 */
export function parseVersion(value) {
  const match = typeof value === 'string' && pattern.exec(value);
  if (!match) throw new Error('Invalid release version');
  return { numbers: match.slice(1, 4).map(BigInt), rc: match[4] === undefined ? null : BigInt(match[4]) };
}

/** compareVersions 比较稳定版本和 rc 的 SemVer 顺序。
 * Args:
 *   left: 候选版本。
 *   right: 已有版本。
 * Returns:
 *   候选较小为 -1，相同为 0，较大为 1。
 * Errors:
 *   版本不符合发行契约时返回异常。
 */
export function compareVersions(left, right) {
  const a = parseVersion(left);
  const b = parseVersion(right);
  for (let i = 0; i < 3; i++) {
    if (a.numbers[i] !== b.numbers[i]) return a.numbers[i] > b.numbers[i] ? 1 : -1;
  }
  if (a.rc === b.rc) return 0;
  if (a.rc === null) return 1;
  if (b.rc === null) return -1;
  return a.rc > b.rc ? 1 : -1;
}

/** validateManifest 阻止占位包、未授权许可和 tag/版本不一致的发行。
 * Args:
 *   manifest: 待发行 npm 包说明书。
 *   tag: 可选待校验的 Git tag。
 * Returns:
 *   校验后的版本及稳定/预发布通道。
 * Errors:
 *   未准备好发行的包返回异常，不改变文件或云端状态。
 */
export function validateManifest(manifest, tag) {
  const version = manifest?.version;
  const parsed = parseVersion(version);
  if (version === '0.0.0') throw new Error('Release version is still a placeholder');
  if (manifest.private !== false) throw new Error('Release package is private');
  if (typeof manifest.name !== 'string' || !/^@[a-z0-9-]+\/[a-z0-9-]+$/.test(manifest.name)) throw new Error('Invalid npm package name');
  if (!manifest.license || manifest.license === 'UNLICENSED') throw new Error('Release license is not approved');
  if (tag !== undefined && tag !== 'v' + version) throw new Error('Release tag differs from package version');
  return { version, tag: 'v' + version, channel: parsed.rc === null ? 'latest' : 'rc' };
}

/** planRelease 仅依据 main 前后 manifest 的版本变化确定是否发行。
 * Args:
 *   current: 已合入 main 的 manifest。
 *   previous: push 前的 manifest；首次引入时为 null。
 * Returns:
 *   不发行标记，或经过校验的发行身份。
 * Errors:
 *   不可发行或倒退的版本返回异常。
 */
export function planRelease(current, previous) {
  if (current.version === previous?.version) return { release: false };
  if (previous && compareVersions(current.version, previous.version) <= 0) throw new Error('Release version must increase');
  if (current.version === '0.0.0') return { release: false };
  const identity = validateManifest(current);
  return { release: true, ...identity };
}
