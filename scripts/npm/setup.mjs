import { readFileSync, lstatSync, existsSync } from 'node:fs';
import { homedir } from 'node:os';
import { dirname, isAbsolute, join, parse, resolve } from 'node:path';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { installBinary, packageRoot } from './install.mjs';
import { prepareLegacySkillCleanup, removeLegacySkillLinks, parseAgents, validateBundledSkill } from './skills.mjs';
import { preparePluginIntegration, attachPluginIntegration, installNativePlugin, runNative } from './plugin.mjs';

const exec = promisify(execFile);
const officialPackage = '@validflow/ccneed-cli';
const releaseVersion = /^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-rc\.(0|[1-9][0-9]*))?$/;

/** releaseManifest 只接受已启用发行的官方正式包或 RC 包。
 * Args:
 *   root: 本轮 npm 包路径。
 * Returns:
 *   已确认的官方包身份。
 * Errors:
 *   私有、占位、未知身份或未授权许可禁止安装。
 */
function releaseManifest(root) {
  const manifest = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));
  if (manifest.name !== officialPackage || manifest.private !== false || manifest.license === 'UNLICENSED' || !manifest.license || !releaseVersion.test(manifest.version) || manifest.version === '0.0.0') {
    throw new Error('Development package is private or unpublished; unified global installation is disabled');
  }
  return manifest;
}

/** runNpm 执行固定 npm 参数，不启动 shell 或向 npm 传入产品凭证。
 * Args:
 *   args: 安装器构造的 npm 参数。
 * Returns:
 *   受限长度的标准输出。
 * Errors:
 *   npm 不可用、超时或失败返回脱敏异常。
 */
export async function runNpm(args) {
  let command = 'npm';
  let parameters = args;
  if (process.platform === 'win32') {
    const candidates = [process.env.npm_execpath, join(dirname(process.execPath), 'node_modules', 'npm', 'bin', 'npm-cli.js')];
    const script = candidates.find(path => path && isAbsolute(path) && path.endsWith('npm-cli.js') && existsSync(path));
    if (!script) throw new Error('npm launcher unavailable; run the installer through npx');
    command = process.execPath;
    parameters = [script, ...args];
  }
  try {
    const result = await exec(command, parameters, { timeout: 180_000, maxBuffer: 1024 * 1024, windowsHide: true, shell: false });
    return result.stdout;
  } catch { throw new Error('npm operation failed; installation is not complete. Check npm access and retry the same installer.'); }
}

/** verifyInstalledBinary 校验包内程序摘要和程序实际版本。
 * Args:
 *   root: 安装后的全局 npm 包路径。
 *   version: 安装入口选择的精确版本。
 * Returns:
 *   已验证的程序路径。
 * Errors:
 *   安装、版本或程序启动失败返回异常。
 */
export async function verifyInstalledBinary(root, version) {
  const binary = await installBinary({ root });
  const result = await exec(binary, ['version', '--no-update-check'], { timeout: 10_000, maxBuffer: 64 * 1024, windowsHide: true, shell: false });
  if (JSON.parse(result.stdout).version !== version) throw new Error('Installed executable version mismatch');
  return binary;
}

/** existingManifest 读取当前安装版本，不将损坏的包误认为尚未安装。
 * Args:
 *   root: 全局官方包路径。
 * Returns:
 *   已有包说明书或 null。
 * Errors:
 *   未知包身份或损坏说明书返回异常。
 */
function existingManifest(root) {
  try { lstatSync(root); } catch (error) { if (error.code === 'ENOENT') return null; throw error; }
  return releaseManifest(root);
}

/** newerVersion 按 SemVer 比较已装正式版与 RC，禁止旧安装入口静默降级。
 * Args:
 *   current: 当前官方已安装版本。
 *   desired: 请求安装的官方正式版或 RC 版本。
 * Returns:
 *   当前版本是否更高。
 */
function newerVersion(current, desired) {
  const left = releaseVersion.exec(current);
  const right = releaseVersion.exec(desired);
  for (let index = 1; index <= 3; index++) {
    if (BigInt(left[index]) !== BigInt(right[index])) return BigInt(left[index]) > BigInt(right[index]);
  }
  if (left[4] === undefined) return right[4] !== undefined;
  if (right[4] === undefined) return false;
  return BigInt(left[4]) > BigInt(right[4]);
}

/** customNativeProfile 识别无法与默认权限配置一起自动安装的宿主 profile。
 * Args:
 *   host: 本次选择的宿主。
 *   home: 安装器使用的用户目录。
 *   env: 原生进程实际继承的环境，测试可随 native 边界注入。
 * Returns:
 *   非默认 profile 的环境变量名，默认 profile 返回 null。
 */
function customNativeProfile(host, home, env) {
  const variable = { codex: 'CODEX_HOME', claude: 'CLAUDE_CONFIG_DIR' }[host];
  const value = variable && env?.[variable];
  if (!value) return null;
  if (typeof value !== 'string') return variable;
  const clean = path => path.replace(/^\\\\\?\\UNC\\/i, '\\\\').replace(/^\\\\\?\\/, '');
  if (!isAbsolute(clean(value))) return variable;
  const actual = resolve(clean(value));
  const expected = resolve(join(home, host === 'codex' ? '.codex' : '.claude'));
  const same = process.platform === 'win32' ? actual.toLowerCase() === expected.toLowerCase() : actual === expected;
  return same ? null : variable;
}

/** installIntegration 全局安装精确版本 CLI，再逐宿主验证原生插件生命周期。
 * Args:
 *   options: 包来源、宿主、用户目录和可替换的 npm/程序/native(host,args)/env 边界。
 * Returns:
 *   CLI 版本、各宿主实际版本及信任、重启或自定义 profile 的手动设置要求。
 * Errors:
 *   npm/CLI 完整性错误直接失败；原生宿主失败返回真实的逐宿主部分结果。
 */
export async function installIntegration({ root = packageRoot, agents, home = homedir(), npm = runNpm, verify = verifyInstalledBinary, native = runNative, env = process.env }) {
  const desired = releaseManifest(root);
  const selected = parseAgents(agents);
  // The real runner inherits process.env; injected runners may supply their
  // own environment snapshot without changing the installer's process.
  const nativeEnv = native === runNative ? process.env : env;
  const customProfiles = new Map(selected.map(host => [host, customNativeProfile(host, home, nativeEnv)]));
  const automatic = selected.filter(host => !customProfiles.get(host));
  if (native === runNative && resolve(home) !== resolve(homedir()) && automatic.some(host => host !== 'workbuddy')) throw new Error('A custom home requires an injected native process boundary');
  validateBundledSkill(root);
  const globalRoot = (await npm(['root', '--global'])).trim();
  if (!isAbsolute(globalRoot) || globalRoot === parse(globalRoot).root || /[\r\n]/.test(globalRoot)) throw new Error('Invalid global npm directory');
  const globalPackage = join(globalRoot, '@validflow', 'ccneed-cli');
  prepareLegacySkillCleanup(home, automatic, globalPackage);
  preparePluginIntegration(home, automatic, globalPackage);
  const current = existingManifest(globalPackage);
  if (current) validateBundledSkill(globalPackage);
  if (current && newerVersion(current.version, desired.version)) throw new Error('Refusing to downgrade; use the current official installer');
  if (!current || current.version !== desired.version) {
    await npm(['install', '--global', officialPackage + '@' + desired.version, '--registry=https://registry.npmjs.org', '--no-audit', '--no-fund']);
  }
  if ((await npm(['root', '--global'])).trim() !== globalRoot) throw new Error('Global npm directory changed during installation');
  if (releaseManifest(globalPackage).version !== desired.version) throw new Error('Global package version mismatch');
  validateBundledSkill(globalPackage);
  await verify(globalPackage, desired.version);
  const plugins = [];
  for (const host of selected) {
    const variable = customProfiles.get(host);
    if (variable) {
      plugins.push({ host, plugin_id: 'ccneed@ccneed', requested_version: desired.version, version: null, status: 'action_required', error: 'custom_profile', next_steps: [variable + ' selects a nondefault native profile; automatic registration, default-profile rules and legacy cleanup were not performed for this host', 'Use that profile’s native plugin manager to register ' + globalPackage + ' and install/enable ccneed@ccneed version ' + desired.version, 'Set up the required native permission rules in the same active profile, review and trust the hooks, and restart the host before collection'] });
      continue;
    }
    const result = await installNativePlugin(native, host, globalPackage, desired.version);
    if (result.status === 'installed' || (host === 'workbuddy' && result.status === 'action_required')) {
      try {
        // Native managers can update settings during registration. Re-read
        // rather than overwriting their enabledPlugins or marketplace state.
        result.configuration_targets = attachPluginIntegration(preparePluginIntegration(home, [host], globalPackage));
        result.removed_legacy_skill_links = removeLegacySkillLinks(prepareLegacySkillCleanup(home, [host], globalPackage));
      } catch {
        result.status = 'action_required';
        result.error = 'legacy_integration_cleanup_required';
        result.next_steps = ['Native registration or manual import is not sufficient yet: review the retained custom or concurrently changed legacy configuration, then retry the installer'];
      }
    }
    plugins.push(result);
  }
  return { status: plugins.every(plugin => plugin.status === 'installed') ? 'installed' : 'action_required', version: desired.version, agents: selected, plugins, restart_required: true, hook_trust_required: true, manual_approval_required: true, auth: 'not_started' };
}

/** installerMain 提供安装与更新共用的明确入口，帮助和参数错误不触发网络。
 * Args:
 *   args: install 子命令后的参数。
 * Returns:
 *   完成后的安装 JSON，或离线帮助文本。
 * Errors:
 *   未指定宿主或非法参数返回异常，不自动选择全部宿主或登录。
 */
export async function installerMain(args) {
  if (args.length === 1 && ['--help', '-h'].includes(args[0])) {
    return 'Install or update the global CLI and register same-version native plugins.\nUsage: npx @validflow/ccneed-cli@latest install --agent codex,workbuddy\nRC: npx @validflow/ccneed-cli@next install --agent codex,workbuddy\nHosts: codex, claude; WorkBuddy requires native desktop import. Check each host result, restart, trust hooks, and keep manual approvals enabled.\n';
  }
  if (args.length !== 2 || args[0] !== '--agent') throw new Error('Use install --agent codex,workbuddy,claude; see install --help');
  return JSON.stringify(await installIntegration({ agents: args[1] })) + '\n';
}
