import { createHash } from 'node:crypto';
import { mkdirSync, readFileSync, writeFileSync, copyFileSync, cpSync, lstatSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { assetName } from '../npm/install.mjs';
import { validateManifest } from './version.mjs';

/** targets 定义安装器和发行构建共同支持的六个平台。
 * Args:
 *   version: 经过校验的发行号。
 * Returns:
 *   Go OS/arch 与对应 asset 名称。
 * Errors:
 *   非法版本返回异常。
 */
export function targets(version) {
  return ['linux', 'darwin', 'windows'].flatMap((os) => ['amd64', 'arm64'].map((arch) => ({
    os, arch, asset: assetName(version, os === 'windows' ? 'win32' : os, arch === 'amd64' ? 'x64' : arch),
  })));
}

/** sha256 校验本地产物，不读取认证信息。
 * Args:
 *   path: 明确的产物路径。
 * Returns:
 *   SHA-256 十六进制摘要。
 * Errors:
 *   文件缺失或读取失败返回异常。
 */
export function sha256(path) { return createHash('sha256').update(readFileSync(path)).digest('hex'); }

/** rejectLinks 防止宿主 Skill 含有越界符号链接。
 * Args:
 *   path: 当前发行源的 Skill 目录或文件。
 * Errors:
 *   任意符号链接返回异常，禁止解引用到包外。
 */
function rejectLinks(path) {
  const stat = lstatSync(path);
  if (stat.isSymbolicLink()) throw new Error('Skill cannot contain symbolic links');
  if (stat.isDirectory()) for (const name of readdirSync(path)) rejectLinks(join(path, name));
}

/** stagePackage 将同一源码版本的 npm 包装、校验和与 Skill 装配到 dist/npm。
 * Args:
 *   root: 已审查的源码目录。
 *   out: 该目录下本轮构建输出目录。
 *   manifest: 本轮发行说明书。
 * Returns:
 *   待执行 npm pack 的目录。
 * Errors:
 *   二进制、许可文件或 Skill 缺失以及版本不一致时返回异常。
 */
export function stagePackage(root, out, manifest) {
  validateManifest(manifest);
  const source = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));
  if (source.version !== manifest.version || source.name !== manifest.name) throw new Error('Package staging version mismatch');
  const skill = join(root, 'skills', 'ccneed');
  rejectLinks(skill);
  if (!readFileSync(join(skill, 'SKILL.md'), 'utf8').trim()) throw new Error('Release requires a usable Skill');
  const assets = Object.fromEntries(targets(manifest.version).map(({ asset }) => [asset, sha256(join(out, asset))]));
  const checksums = JSON.stringify({ version: manifest.version, assets }, null, 2) + '\n';
  writeFileSync(join(out, 'checksums.json'), checksums);
  const destination = join(out, 'npm');
  mkdirSync(join(destination, 'scripts', 'npm'), { recursive: true });
  writeFileSync(join(destination, 'package.json'), JSON.stringify(manifest, null, 2) + '\n');
  writeFileSync(join(destination, 'checksums.json'), checksums);
  for (const file of ['install.mjs', 'run.mjs']) copyFileSync(join(root, 'scripts', 'npm', file), join(destination, 'scripts', 'npm', file));
  for (const file of ['README.md', 'LICENSE']) copyFileSync(join(root, file), join(destination, file));
  cpSync(skill, join(destination, 'skills', 'ccneed'), { recursive: true });
  return destination;
}

/** verifyArtifacts 校验安装包版本、六个平台文件和全部摘要。
 * Args:
 *   out: 本轮产物目录。
 * Returns:
 *   校验成功的二进制文件名数组。
 * Errors:
 *   版本、产物或校验和不匹配时返回异常，不允许发布部分平台。
 */
export function verifyArtifacts(out) {
  const manifest = JSON.parse(readFileSync(join(out, 'npm', 'package.json'), 'utf8'));
  validateManifest(manifest);
  const checksums = JSON.parse(readFileSync(join(out, 'checksums.json'), 'utf8'));
  if (checksums.version !== manifest.version || readFileSync(join(out, 'npm', 'checksums.json'), 'utf8') !== readFileSync(join(out, 'checksums.json'), 'utf8')) throw new Error('Artifact checksum version mismatch');
  const names = targets(manifest.version).map(({ asset }) => asset);
  if (Object.keys(checksums.assets).length !== names.length) throw new Error('Incomplete artifact checksums');
  for (const name of names) if (checksums.assets[name] !== sha256(join(out, name))) throw new Error('Artifact checksum mismatch');
  return names;
}
