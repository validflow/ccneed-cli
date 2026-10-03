import { readFileSync, lstatSync, existsSync } from 'node:fs';
import { homedir } from 'node:os';
import { dirname, isAbsolute, join, parse } from 'node:path';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { installBinary, packageRoot } from './install.mjs';
import { prepareSkillLinks, attachSkillLinks, parseAgents, validateBundledSkill } from './skills.mjs';

const exec = promisify(execFile);
const officialPackage = '@validflow/ccneed-cli';
const stableVersion = /^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$/;

/** releaseManifest 只接受已启用发行的官方稳定包。
 * Args:
 *   root: 本轮 npm 包路径。
 * Returns:
 *   已确认的官方包身份。
 * Errors:
 *   私有、占位、未知身份或未授权许可禁止安装。
 */
function releaseManifest(root) {
  const manifest = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));
  if (manifest.name !== officialPackage || manifest.private !== false || manifest.license === 'UNLICENSED' || !manifest.license || !stableVersion.test(manifest.version) || manifest.version === '0.0.0') {
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

/** newerVersion 检查已装版本，禁止旧安装入口静默降级。
 * Args:
 *   current: 当前稳定版本。
 *   desired: 请求安装的稳定版本。
 * Returns:
 *   当前版本是否更高。
 */
function newerVersion(current, desired) {
  const left = current.split('.').map(BigInt);
  const right = desired.split('.').map(BigInt);
  for (let index = 0; index < 3; index++) if (left[index] !== right[index]) return left[index] > right[index];
  return false;
}

/** installIntegration 全局安装精确版本 CLI，并将所选宿主指向同包技能。
 * Args:
 *   options: 包来源、明确宿主、用户目录及可替换的 npm/程序验证边界。
 * Returns:
 *   版本、宿主路径和需要重新加载的真实安装结果。
 * Errors:
 *   自定义技能冲突在 npm 写入前失败；npm/版本/验证失败不会报告成功。
 */
export async function installIntegration({ root = packageRoot, agents, home = homedir(), npm = runNpm, verify = verifyInstalledBinary }) {
  const desired = releaseManifest(root);
  const selected = parseAgents(agents);
  validateBundledSkill(root);
  const globalRoot = (await npm(['root', '--global'])).trim();
  if (!isAbsolute(globalRoot) || globalRoot === parse(globalRoot).root || /[\r\n]/.test(globalRoot)) throw new Error('Invalid global npm directory');
  const globalPackage = join(globalRoot, '@validflow', 'ccneed-cli');
  const plans = prepareSkillLinks(home, selected, globalPackage);
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
  const paths = attachSkillLinks(plans);
  return { status: 'installed', version: desired.version, agents: selected, skill_targets: paths, restart_required: true, auth: 'not_started' };
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
    return 'Install or update CLI + same-version Skill globally.\nUsage: npx @validflow/ccneed-cli@latest install --agent codex,workbuddy\nHosts: codex, workbuddy, claude. Restart selected hosts after updating.\n';
  }
  if (args.length !== 2 || args[0] !== '--agent') throw new Error('Use install --agent codex,workbuddy,claude; see install --help');
  return JSON.stringify(await installIntegration({ agents: args[1] })) + '\n';
}
