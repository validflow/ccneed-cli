import { execFileSync } from 'node:child_process';

/** git 执行有超时的只读 Git 查询，不经过 shell。
 * Args:
 *   args: Git 参数数组；只由可信发行控制代码提供。
 *   cwd: 明确的仓库目录。
 * Returns:
 *   去掉尾部换行的标准输出。
 * Errors:
 *   Git 返回非零状态或超时时返回异常。
 */
export function git(args, cwd = process.cwd()) {
  return execFileSync('git', args, { cwd, encoding: 'utf8', timeout: 30_000, stdio: ['ignore', 'pipe', 'pipe'] }).trim();
}

/** manifestAt 读取已存在提交的版本文件，区分“旧提交无文件”与 Git 失败。
 * Args:
 *   sha: 40 位提交 SHA。
 *   cwd: 仓库目录。
 * Returns:
 *   提交中的 manifest，或不存在文件时的 null。
 * Errors:
 *   非法 SHA、缺失提交、损坏 JSON 或 Git 查询失败返回异常。
 */
export function manifestAt(sha, cwd = process.cwd()) {
  if (!/^[a-f0-9]{40}$/.test(sha)) throw new Error('Invalid main revision');
  if (/^0{40}$/.test(sha)) return null;
  git(['cat-file', '-e', sha + '^{commit}'], cwd);
  const files = git(['ls-tree', '--name-only', sha, '--', 'package.json'], cwd);
  if (!files) return null;
  return JSON.parse(git(['show', sha + ':package.json'], cwd));
}
