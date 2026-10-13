import { readFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
import { validateManifest } from './version.mjs';

/** releaseNotification 构造稳定版发行回执，不把 webhook 凭证写入负载。
 * Args:
 *   manifest: 当前 main 的 npm 包说明书。
 *   env: GitHub Actions 的发行身份。
 * Returns:
 *   RC 返回 null；正式版返回用于通知的事件与幂等键。
 * Errors:
 *   版本、仓库或提交身份不一致时返回异常。
 */
export function releaseNotification(manifest, env) {
  const identity = validateManifest(manifest, env.RELEASE_TAG);
  if (identity.channel !== 'latest') return null;
  if (env.GITHUB_REPOSITORY !== 'validflow/ccneed-cli' || !/^[a-f0-9]{40}$/.test(env.GITHUB_SHA ?? '') || !/^[1-9][0-9]*$/.test(env.GITHUB_RUN_ID ?? '')) {
    throw new Error('Invalid GitHub release identity');
  }
  return {
    key: `ccneed-cli:${identity.tag}`,
    payload: {
      event: 'ccneed-cli.release.verified',
      repository: env.GITHUB_REPOSITORY,
      version: identity.version,
      tag: identity.tag,
      channel: identity.channel,
      commit: env.GITHUB_SHA,
      release_url: `https://github.com/validflow/ccneed-cli/releases/tag/${identity.tag}`,
      npm_url: `https://www.npmjs.com/package/@validflow/ccneed-cli/v/${identity.version}`,
      run_url: `https://github.com/validflow/ccneed-cli/actions/runs/${env.GITHUB_RUN_ID}`,
    },
  };
}

/** notifyMultica 将已验证的稳定版结果交给 Multica，由 Autopilot 核验并发群卡。
 * Args:
 *   env: GitHub Actions 环境变量，包括保存在 Secret 中的 webhook URL。
 *   request: 可注入的 HTTP 请求函数。
 *   manifest: 当前 main 的 npm 包说明书。
 * Returns:
 *   skipped、accepted 或 duplicate；accepted 只代表 Multica 接收，不代表群消息已送达。
 * Errors:
 *   缺少凭证、网络错误或 Multica 拒绝时返回不含凭证的异常。
 */
export async function notifyMultica(env, request, manifest) {
  const notification = releaseNotification(manifest, env);
  if (!notification) return 'skipped';
  let webhook;
  try { webhook = new URL(env.MULTICA_CLI_RELEASE_WEBHOOK_URL); } catch { throw new Error('Missing or invalid Multica webhook secret'); }
  if (webhook.protocol !== 'https:' || webhook.hostname !== 'multica.ai' || !/^\/api\/webhooks\/autopilots\/[A-Za-z0-9_-]+$/.test(webhook.pathname) || webhook.search || webhook.hash || webhook.username || webhook.password) {
    throw new Error('Invalid Multica webhook destination');
  }
  let response;
  try {
    response = await request(webhook.href, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Idempotency-Key': notification.key },
      body: JSON.stringify(notification.payload),
      redirect: 'error',
      signal: AbortSignal.timeout(15_000),
    });
  } catch { throw new Error('Multica webhook request failed'); }
  if (!response.ok) throw new Error(`Multica webhook returned HTTP ${response.status}`);
  let result;
  try { result = await response.json(); } catch { throw new Error('Invalid Multica webhook response'); }
  if (!['accepted', 'duplicate'].includes(result?.status)) throw new Error('Multica did not start the notification run');
  return result.status;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const manifest = JSON.parse(readFileSync('package.json', 'utf8'));
  try {
    const status = await notifyMultica(process.env, fetch, manifest);
    console.log(`Multica release notification ${status} for ${process.env.RELEASE_TAG}`);
  } catch (error) {
    console.error(`Multica release notification failed: ${error.message}`);
    process.exitCode = 1;
  }
}
