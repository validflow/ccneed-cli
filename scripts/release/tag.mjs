import { pathToFileURL } from 'node:url';
import { readFileSync } from 'node:fs';
import { git } from './git.mjs';
import { validateManifest } from './version.mjs';

/** ensureTag 创建精确提交的轻量 tag，重试仅接受已经指向同一提交的 tag。
 * Args:
 *   api: 只允许固定仓库 REST 路径的 API 调用函数。
 *   tag: 已通过 manifest 校验的 tag。
 *   sha: 本次 main push 的精确提交。
 * Returns:
 *   新建或复用标记。
 * Errors:
 *   tag 已指向别的提交、权限失败或创建失败时返回异常，绝不移动 tag。
 */
export async function ensureTag(api, tag, sha) {
  const verify = (ref) => {
    if (ref.object?.type !== 'commit' || ref.object.sha !== sha) throw new Error('Existing tag points to another commit');
  };
  const existing = await api('GET', '/git/ref/tags/' + tag, null, [200, 404]);
  if (existing.status === 200) { verify(existing.body); return 'reused'; }
  const created = await api('POST', '/git/refs', { ref: 'refs/tags/' + tag, sha }, [201, 422]);
  if (created.status === 201) { verify(created.body); return 'created'; }
  const raced = await api('GET', '/git/ref/tags/' + tag);
  verify(raced.body);
  return 'reused';
}

/** githubAPI 为固定仓库创建有限时且禁止重定向的发行请求函数。
 * Args:
 *   token: 当前 Actions 的短期 GitHub token，不用于产品 API。
 * Returns:
 *   接受 method/path/body/允许状态码的请求函数。
 * Errors:
 *   网络或非允许状态码返回安全错误，不回显响应或 token。
 */
export function githubAPI(token) {
  return async (method, path, body = null, statuses = [200]) => {
    if (!token || !/^\/(git|releases)(\/|\?)/.test(path)) throw new Error('Invalid release API request');
    const response = await fetch('https://api.github.com/repos/validflow/ccneed-cli' + path, {
      method, redirect: 'error', signal: AbortSignal.timeout(30_000),
      headers: { Authorization: 'Bearer ' + token, Accept: 'application/vnd.github+json', 'Content-Type': 'application/json' },
      body: body === null ? undefined : JSON.stringify(body),
    });
    if (!statuses.includes(response.status)) throw new Error('Release API failed with status ' + response.status);
    return { status: response.status, body: response.status === 204 ? null : await response.json() };
  };
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    const manifest = JSON.parse(readFileSync('package.json', 'utf8'));
    const { tag } = validateManifest(manifest, process.env.RELEASE_TAG);
    const sha = process.env.GITHUB_SHA;
    if (!/^[a-f0-9]{40}$/.test(sha) || git(['rev-parse', 'HEAD']) !== sha || git(['status', '--porcelain'])) throw new Error('Tag requires a clean exact main checkout');
    await ensureTag(githubAPI(process.env.GH_TOKEN), tag, sha);
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
