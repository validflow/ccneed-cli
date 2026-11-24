import { cpSync, copyFileSync, lstatSync, mkdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fingerprintSkill } from './skills.mjs';

export const additionalHosts = ['openclaw', 'pi', 'zcode', 'qoder', 'trae', 'dsh'];
export const additionalPayloads = ['plugins', 'openclaw.plugin.json', 'marketplace.json', '.zcode-plugin', '.qoder-plugin'];

/** payloadText 读取固定包内入口，不接受链接或空载荷。
 * Args:
 *   root: 已验证的包根目录。
 *   path: 由插件实现固定的相对路径。
 * Returns:
 *   完整 UTF-8 文件内容。
 * Errors:
 *   缺失、链接、超大或空文件时停止注册。
 */
function payloadText(root, path) {
  const filename = join(root, path);
  const stat = lstatSync(filename);
  if (!stat.isFile() || stat.isSymbolicLink() || stat.size > 1024 * 1024) throw new Error('Invalid native plugin payload');
  const text = readFileSync(filename, 'utf8');
  if (!text.trim()) throw new Error('Empty native plugin payload');
  return text;
}

/** validateAdditionalPayload 校验额外原生插件的版本与必要运行入口。
 * Args:
 *   root: 同一版本的完整 npm 包目录。
 *   host: 明确选择的新宿主。
 *   version: 安装器核实的 CLI 版本。
 *   options: 成品检查时要求 Trae 子插件包含同源完整资源。
 * Errors:
 *   版本或原生入口不一致时拒绝注册。
 */
export function validateAdditionalPayload(root, host, version, { requireProjected = false } = {}) {
  if (!additionalHosts.includes(host)) throw new Error('Unknown native plugin host');
  const pkg = JSON.parse(payloadText(root, 'package.json'));
  if (pkg.version !== version) throw new Error('Native plugin version mismatch');
  if (host === 'openclaw') {
    const manifest = JSON.parse(payloadText(root, 'openclaw.plugin.json'));
    if (manifest.id !== 'ccneed' || manifest.version !== version || !manifest.skills?.includes('./skills') || !pkg.openclaw?.extensions?.includes('./plugins/openclaw.mjs')) throw new Error('OpenClaw plugin version or entry mismatch');
    payloadText(root, 'plugins/openclaw.mjs');
  } else if (host === 'pi') {
    if (!pkg.pi?.extensions?.includes('./plugins/pi.mjs') || !pkg.pi?.skills?.includes('./skills')) throw new Error('Invalid Pi package entry');
    payloadText(root, 'plugins/pi.mjs');
  } else if (host === 'dsh') {
    if (pkg.dsh?.bundle?.patch !== './plugins/dsh/cordis.patch.yml') throw new Error('Invalid DSH bundle entry');
    payloadText(root, 'plugins/dsh/cordis.patch.yml');
    payloadText(root, 'plugins/dsh/index.mjs');
  } else {
    const manifestPath = host === 'trae' ? 'plugins/trae/.codex-plugin/plugin.json' : '.' + host + '-plugin/plugin.json';
    const manifest = JSON.parse(payloadText(root, manifestPath));
    if (manifest.name !== 'ccneed' || manifest.version !== version || manifest.skills !== './skills' || manifest.hooks !== './hooks/' + host + '.json') throw new Error('Native plugin version or entry mismatch');
    const config = JSON.parse(payloadText(root, 'hooks/' + host + '.json'));
    if (!['SessionStart', 'PreToolUse', 'PostToolUse'].every(event => config.hooks?.[event]?.length)) throw new Error('Missing native collection hooks');
    if (host === 'zcode') {
      const catalog = JSON.parse(payloadText(root, 'marketplace.json'));
      if (catalog.name !== 'ccneed' || !catalog.plugins?.some(plugin => plugin.name === 'ccneed' && plugin.source === './' && plugin.version === version)) throw new Error('Invalid ZCode marketplace');
    }
    if (host === 'trae' && requireProjected) {
      payloadText(root, 'plugins/trae/hooks/acquisition.mjs');
      payloadText(root, 'plugins/trae/hooks/trae.json');
      if (fingerprintSkill(join(root, 'plugins/trae/skills/ccneed')) !== fingerprintSkill(join(root, 'skills/ccneed'))) throw new Error('Trae plugin Skill differs from the CLI release');
    }
  }
}

/** stageTraePayload 装配使用 Codex manifest 格式的独立 Trae 原生插件。
 * Args:
 *   root: 同版本共享技能与 hook 的源码目录。
 *   destination: 已装配 npm 包的根目录。
 * Errors:
 *   任一共享资源缺失时禁止交付不完整子插件。
 */
export function stageTraePayload(root, destination) {
  const plugin = join(destination, 'plugins/trae');
  mkdirSync(join(plugin, 'hooks'), { recursive: true });
  cpSync(join(root, 'skills/ccneed'), join(plugin, 'skills/ccneed'), { recursive: true });
  for (const file of ['acquisition.mjs', 'trae.json']) copyFileSync(join(root, 'hooks', file), join(plugin, 'hooks', file));
}

/** additionalPluginInstructions 提供原生管理器的导入与整包更新步骤。
 * Args:
 *   host: 用户选择的宿主。
 *   root: 全局 CLI 同版本包目录。
 *   version: 用户需要在宿主核对的版本。
 * Returns:
 *   明确的待完成步骤；不将文件存在误报为启用成功。
 */
export function additionalPluginInstructions(host, root, version) {
  const path = root.replaceAll('\\', '/');
  const common = ['Verify CCNeed version ' + version + ' in the native plugin manager, enable it, review hooks and restart/reload the host. Repeat the native update/import after each CLI update; npm alone does not replace native caches.'];
  if (host === 'openclaw') return ['Use OpenClaw 2026.11.4 or a compatible newer release. In its native manager run: openclaw plugins install --link "' + path + '"', 'Review and enable ccneed with openclaw plugins enable ccneed; inspect the source/version with openclaw plugins inspect ccneed --runtime --json. Older runtimes do not support the required native per-call approvals.', ...common];
  if (host === 'pi') return ['In Pi, run: pi install "' + path + '"', 'Use pi list and pi config to verify and enable the ccneed package resources, then /reload. Collection requires the interactive native confirmation UI; headless runs cannot approve paid starts.', ...common];
  if (host === 'zcode') return ['In ZCode Settings > Plugins, add the local marketplace file "' + path + '/marketplace.json", then install and enable ccneed.', ...common];
  if (host === 'qoder') return ['Use the Qoder native manager: qoder plugins install "' + path + '"', 'Verify ccneed@local in /plugins, enable it and /plugins reload. Reimport/update the complete plugin when updating the CLI.', 'For interactive per-call confirmation, add these entries to permissions.ask in the active Qoder settings: ' + ['ccneed', 'ccneed.cmd', 'ccneed.exe'].flatMap(binary => ['kb start', 'tasks reply'].map(command => `Bash(${binary} ${command}:*)`)).join(', ') + '. Qoder applies its active mode and rules; the installer does not change them.', ...common];
  if (host === 'dsh') return ['Use DeepSeek Harness 0.2.2-rc.3 or a compatible 0.2.x release with Node.js 24.9+. The stale npm latest tag is not supported. In its native manager select the running profile: dsh plugin --profile <name> add "' + path + '"', 'Verify the CCNeed layer with dsh --profile <name> --dump-config and the Plugins page. Native tool approvals remain governed by the active sandbox and approval policy; removal uses dsh plugin --profile <name> remove @validflow/ccneed-cli.', ...common];
  return ['In Trae CLI 2.0 (verified native 0.215.0), run: traecli plugin install "' + path + '/plugins/trae" --name ccneed', 'Verify traecli plugin validate ccneed reports 1 Skill and 3 hooks. After updating the CLI, run traecli plugin upgrade ccneed to replace the active native cache, then reload. Inactive versions remain owned by the native manager. This integration targets Trae CLI, not Trae IDE.', ...common];
}
