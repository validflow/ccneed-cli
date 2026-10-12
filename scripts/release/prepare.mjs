import { readFileSync, appendFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
import { git, manifestAt } from './git.mjs';
import { planRelease } from './version.mjs';

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

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    const plan = prepare(process.env.BEFORE_SHA, process.env.GITHUB_SHA);
    if (!process.env.GITHUB_OUTPUT) throw new Error('Missing CI output file');
    appendFileSync(process.env.GITHUB_OUTPUT, Object.entries(plan).map(([key, value]) => key + '=' + value).join('\n') + '\n');
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
