import { readFileSync, appendFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
import { git, manifestAt } from './git.mjs';
import { planRelease, validateManifest } from './version.mjs';

/** prepare 将 main push 的版本变更投影为 CI 输出，不创建 tag。
 * Args:
 *   before: push 前提交 SHA。
 *   expected: 本次受信任 main 提交 SHA。
 *   cwd: 仓库目录。
 * Returns:
 *   发行计划；普通合并返回 release=false。
 * Errors:
 *   提交来源不一致、版本文件损坏或倒退时返回异常。
 */
export function prepare(before, expected, cwd = process.cwd()) {
  if (!/^[a-f0-9]{40}$/.test(expected) || git(['rev-parse', 'HEAD'], cwd) !== expected) throw new Error('Release checkout differs from main push');
  const current = JSON.parse(readFileSync(cwd + '/package.json', 'utf8'));
  return planRelease(current, manifestAt(before, cwd));
}

/** prepareRetry 为同版本失败发行显式重跑完整构建和发布门禁。
 * Args:
 *   requested: 维护者手动输入的已在 main 中的版本号。
 *   ref: 当前 Actions 的完整 Git 分支引用。
 *   expected: 本次 main 提交 SHA。
 *   cwd: 仓库目录。
 * Returns:
 *   经 manifest 校验的发行计划。
 * Errors:
 *   非 main、提交不符或请求版本不等于当前 manifest 时返回异常。
 */
export function prepareRetry(requested, ref, expected, cwd = process.cwd()) {
  if (ref !== 'refs/heads/main') throw new Error('Release retry requires the main branch');
  if (!/^[a-f0-9]{40}$/.test(expected) || git(['rev-parse', 'HEAD'], cwd) !== expected) throw new Error('Release checkout differs from main dispatch');
  const current = JSON.parse(readFileSync(cwd + '/package.json', 'utf8'));
  if (requested !== current.version) throw new Error('Release retry requested version differs from package.json');
  return { release: true, ...validateManifest(current, 'v' + requested) };
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    const plan = process.env.GITHUB_EVENT_NAME === 'workflow_dispatch'
      ? prepareRetry(process.env.RETRY_VERSION, process.env.GITHUB_REF, process.env.GITHUB_SHA)
      : prepare(process.env.BEFORE_SHA, process.env.GITHUB_SHA);
    if (!process.env.GITHUB_OUTPUT) throw new Error('Missing CI output file');
    appendFileSync(process.env.GITHUB_OUTPUT, Object.entries(plan).map(([key, value]) => key + '=' + value).join('\n') + '\n');
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
