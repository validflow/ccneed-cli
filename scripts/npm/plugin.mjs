import { lstatSync, readFileSync, mkdirSync, writeFileSync, renameSync, unlinkSync } from 'node:fs';
import { delimiter, dirname, isAbsolute, join, resolve } from 'node:path';
import { randomUUID } from 'node:crypto';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';

const hostDirs = { codex: '.codex', claude: '.claude', workbuddy: '.workbuddy' };
const commands = ['ccneed', 'ccneed.cmd', 'ccneed.exe'];
const exec = promisify(execFile);

/** fileText 只读取普通配置文件，拒绝链接和过大的配置。
 * Args:
 *   path: 当前选中宿主的配置路径。
 * Returns:
 *   文件原文或 null。
 * Errors:
 *   特殊文件、链接或读取失败返回异常。
 */
function fileText(path) {
  let stat;
  try { stat = lstatSync(path); } catch (error) { if (error.code === 'ENOENT') return null; throw error; }
  if (!stat.isFile() || stat.isSymbolicLink() || stat.size > 1024 * 1024) throw new Error('Plugin config must be a bounded regular file: ' + path);
  return readFileSync(path, 'utf8');
}

/** checkParents 禁止宿主目录通过链接重定向安装写入。
 * Args:
 *   home: 安装目标的明确用户目录。
 *   directory: 当前文件的父目录。
 * Errors:
 *   越界路径、父目录链接或特殊文件返回异常。
 */
function checkParents(home, directory) {
  const root = resolve(home);
  const target = resolve(directory);
  if (target !== root && !target.startsWith(root + (process.platform === 'win32' ? '\\' : '/'))) throw new Error('Plugin target is outside the selected home');
  const paths = [];
  for (let path = target; path !== root; path = dirname(path)) paths.push(path);
  for (const path of [root, ...paths.reverse()]) {
    let stat;
    try { stat = lstatSync(path); } catch (error) { if (error.code === 'ENOENT') continue; throw error; }
    if (!stat.isDirectory() || stat.isSymbolicLink()) throw new Error('Plugin parent must be a plain directory: ' + path);
  }
}

/** hookHandler 重建旧安装器生成的 handler，仅用于识别可迁移配置。
 * Args:
 *   root: 固定全局 npm 包目录。
 *   host: 明确选择的宿主。
 *   node: 旧 handler 中的 Node 路径，不会执行。
 * Returns:
 *   不经动态 shell 插值的命令配置。
 * Errors:
 *   路径包含换行或 NUL 时拒绝生成配置。
 */
function hookHandler(root, host, node = process.execPath) {
  const args = [node, join(root, 'skills', 'ccneed', 'scripts', 'acquisition.mjs'), '--host', host];
  if (args.some(value => /[\r\n\0]/.test(value))) throw new Error('Invalid hook path');
  const posix = args.map(value => "'" + value.replaceAll("'", "'\\''") + "'").join(' ');
  const powershell = '& ' + args.map(value => "'" + value.replaceAll("'", "''") + "'").join(' ');
  if (host === 'codex') return { type: 'command', command: posix, commandWindows: powershell, timeout: 10 };
  return { type: 'command', command: process.platform === 'win32' ? powershell : posix, ...(process.platform === 'win32' ? { shell: 'powershell' } : {}), timeout: 10 };
}

/** sameConfig 比较配置语义，避免 JSON 键序或缩进影响自有配置识别。
 * Args:
 *   left: 待检查配置。
 *   right: 已知安装器配置。
 * Returns:
 *   两份配置是否相同。
 */
function sameConfig(left, right) {
  const canonical = value => Array.isArray(value) ? value.map(canonical) : value && typeof value === 'object' ? Object.fromEntries(Object.keys(value).sort().map(key => [key, canonical(value[key])])) : value;
  return JSON.stringify(canonical(left)) === JSON.stringify(canonical(right));
}

/** ownedLegacyHandler 识别完整旧模板，不把路径包含关键字当作所有权证明。
 * Args:
 *   handler: 用户配置中的一条 handler。
 *   root: 稳定全局包目录。
 *   host: 当前宿主。
 * Returns:
 *   handler 是否为未定制的旧安装器输出。
 */
function ownedLegacyHandler(handler, root, host) {
  if (!handler || typeof handler.command !== 'string') return false;
  const match = /^(?:& )?'((?:[^']|'')*)'\s/.exec(handler.command);
  const node = match?.[1]?.replaceAll("''", "'");
  if (!node || !/(?:^|[\\/])node(?:\.exe)?$/i.test(node)) return false;
  return sameConfig(handler, hookHandler(root, host, node));
}

/** removeOwnedHooks 只清理旧安装器原样生成的全局 hooks，保留定制 handler。
 * Args:
 *   settings: 当前用户配置对象。
 *   root: 稳定全局包目录。
 *   host: 当前宿主。
 * Returns:
 *   是否移除了旧安装器 handler。
 * Errors:
 *   hooks 配置格式无效时停止。
 */
function removeOwnedHooks(settings, root, host) {
  if (settings.hooks === undefined) return false;
  if (!settings.hooks || Array.isArray(settings.hooks) || typeof settings.hooks !== 'object') throw new Error('Plugin hooks must be a JSON object');
  let removed = false;
  for (const event of ['SessionStart', 'PreToolUse', 'PostToolUse']) {
    const entries = settings.hooks[event];
    if (entries === undefined) continue;
    if (!Array.isArray(entries)) throw new Error('Plugin event hooks must be an array');
    settings.hooks[event] = entries.flatMap(entry => {
      const matcher = event === 'SessionStart' ? undefined : 'Bash|PowerShell|exec_command';
      if (!entry || entry.matcher !== matcher || !Array.isArray(entry.hooks) || Object.keys(entry).some(key => !['matcher', 'hooks'].includes(key))) return [entry];
      const hooks = entry.hooks.filter(handler => !ownedLegacyHandler(handler, root, host));
      if (hooks.length === entry.hooks.length) return [entry];
      removed = true;
      return hooks.length ? [{ ...entry, hooks }] : [];
    });
    if (!settings.hooks[event].length) delete settings.hooks[event];
  }
  if (!Object.keys(settings.hooks).length) delete settings.hooks;
  return removed;
}

/** removeOwnedAskRules 在存在旧安装器 hook 证据时清理其通用 ask 规则。
 * Args:
 *   settings: 当前用户配置。
 * Errors:
 *   permissions 配置格式无效时停止。
 */
function removeOwnedAskRules(settings) {
  if (settings.permissions === undefined) return;
  const permissions = settings.permissions;
  if (!permissions || Array.isArray(permissions) || typeof permissions !== 'object') throw new Error('Plugin permissions must be a JSON object');
  if (permissions.ask === undefined) return;
  if (!Array.isArray(permissions.ask) || permissions.ask.some(value => typeof value !== 'string')) throw new Error('Plugin ask rules must be strings');
  const owned = new Set(commands.flatMap(binary => ['Bash', 'PowerShell'].flatMap(tool => [`${tool}(${binary} kb start *)`, `${tool}(${binary} tasks reply *)`])));
  permissions.ask = permissions.ask.filter(value => !owned.has(value));
  if (!permissions.ask.length) delete permissions.ask;
  if (!Object.keys(permissions).length) delete settings.permissions;
}

/** mergeSettings 清理自有旧 hooks，仅为 WorkBuddy 保留独立原生执行规则。
 * Args:
 *   before: 已有配置原文或 null。
 *   root: 固定全局 npm 包目录。
 *   host: 明确选择的宿主。
 * Returns:
 *   合并后的 JSON 配置原文；无变化时保留原文。
 * Errors:
 *   配置格式无效时停止；自定义 handler 原样保留。
 */
function mergeSettings(before, root, host) {
  const settings = before === null ? {} : JSON.parse(before.replace(/^\uFEFF/, ''));
  if (!settings || Array.isArray(settings) || typeof settings !== 'object') throw new Error('Plugin settings must be a JSON object');
  const previous = JSON.stringify(settings);
  if (removeOwnedHooks(settings, root, host) && host !== 'codex') removeOwnedAskRules(settings);
  if (host === 'workbuddy') {
    settings.permissions ??= {};
    if (!settings.permissions || Array.isArray(settings.permissions) || typeof settings.permissions !== 'object') throw new Error('WorkBuddy permissions must be a JSON object');
    const permissionAsk = settings.permissions.ask ?? [];
    if (!Array.isArray(permissionAsk) || permissionAsk.some(value => typeof value !== 'string')) throw new Error('WorkBuddy ask rules must be strings');
    const compatibleAsk = commands.flatMap(binary => ['Bash', 'PowerShell'].flatMap(tool => [`${tool}(${binary} kb start *)`, `${tool}(${binary} tasks reply *)`]));
    settings.permissions.ask = [...new Set([...permissionAsk, ...compatibleAsk])];
    settings.sandbox ??= {};
    if (!settings.sandbox || Array.isArray(settings.sandbox) || typeof settings.sandbox !== 'object') throw new Error('WorkBuddy sandbox must be a JSON object');
    settings.sandbox.commandSafety ??= {};
    const safety = settings.sandbox.commandSafety;
    if (!safety || Array.isArray(safety) || typeof safety !== 'object') throw new Error('WorkBuddy command safety must be a JSON object');
    const ask = safety.ask ?? [];
    if (!Array.isArray(ask) || ask.some(rule => !Array.isArray(rule?.prefix) || !rule.prefix.length || rule.prefix.some(value => typeof value !== 'string'))) throw new Error('WorkBuddy command safety ask rules must contain command prefixes');
    const added = commands.flatMap(binary => [['kb', 'start'], ['tasks', 'reply']].map(words => ({ prefix: [binary, ...words] })));
    safety.ask = [...ask, ...added.filter(rule => !ask.some(existing => JSON.stringify(existing.prefix) === JSON.stringify(rule.prefix)))];
  }
  return previous === JSON.stringify(settings) && before !== null ? before : JSON.stringify(settings, null, 2) + '\n';
}

/** codexRules 生成原生逐次确认策略，保留只读命令自动执行。
 * Returns:
 *   Codex Starlark 规则原文。
 */
export function codexRules() {
  return '# Managed by the CCNeed installer; keep manual approvals enabled.\n' + [['kb', 'start'], ['tasks', 'reply']].map(words =>
    'prefix_rule(pattern = ' + JSON.stringify([commands, ...words]) + ', decision = "prompt", justification = "Confirm this CCNeed operation after reviewing the final collection plan")\n').join('');
}

/** preparePluginIntegration 在 npm 替换包之前检查宿主配置冲突，不写文件。
 * Args:
 *   home: 当前用户的明确目录，测试可注入隔离目录。
 *   agents: 用户明确选择的宿主名称。
 *   root: 固定全局 npm 包目录。
 * Returns:
 *   自有迁移和权限配置目标；不会注册全局 hooks。
 * Errors:
 *   无效路径、已有配置冲突或非法 JSON 返回异常。
 */
export function preparePluginIntegration(home, agents, root) {
  if (!isAbsolute(home) || !isAbsolute(root)) throw new Error('Plugin directories must be absolute');
  return agents.flatMap(host => {
    if (!Object.hasOwn(hostDirs, host)) throw new Error('Unsupported plugin host');
    const directory = join(home, hostDirs[host]);
    const target = join(directory, host === 'codex' ? 'hooks.json' : 'settings.json');
    checkParents(home, dirname(target));
    const before = fileText(target);
    const files = before === null && host !== 'workbuddy' ? [] : [{ home, host, target, before, after: mergeSettings(before, root, host), root }];
    if (host === 'codex') {
      const target = join(directory, 'rules', 'ccneed.rules');
      checkParents(home, dirname(target));
      const before = fileText(target);
      const after = codexRules();
      if (before !== null && before !== after) throw new Error('Existing custom CCNeed rules will not be overwritten');
      files.unshift({ home, host, target, before, after, root });
    }
    return files;
  });
}

/** replaceText 原子替换已核对的配置，避免保留半份 JSON。
 * Args:
 *   path: 明确的宿主配置目标。
 *   text: 完整新配置。
 * Errors:
 *   写入或发布失败时返回异常。
 */
function replaceText(path, text) {
  const temporary = path + '.' + randomUUID() + '.tmp';
  try {
    writeFileSync(temporary, text, { flag: 'wx', mode: 0o600 });
    renameSync(temporary, path);
  } finally {
    try { unlinkSync(temporary); } catch (error) { if (error.code !== 'ENOENT') throw error; }
  }
}

/** attachPluginIntegration 发布明确选择的宿主配置，不修改安全模式或其他设置。
 * Args:
 *   plans: 无副作用预检得到的配置计划。
 * Returns:
 *   实际配置文件路径；仍需宿主信任和重新加载。
 * Errors:
 *   缺失 handler、并发配置修改或写入失败时撤回本次配置写入。
 */
export function attachPluginIntegration(plans) {
  for (const plan of plans) {
    if (!fileText(join(plan.root, 'skills', 'ccneed', 'scripts', 'acquisition.mjs'))?.trim()) throw new Error('Incomplete bundled CCNeed plugin');
    checkParents(plan.home, dirname(plan.target));
    if (fileText(plan.target) !== plan.before) throw new Error('Host settings changed during installation');
  }
  const written = [];
  try {
    for (const plan of plans) {
      checkParents(plan.home, dirname(plan.target));
      mkdirSync(dirname(plan.target), { recursive: true });
      checkParents(plan.home, dirname(plan.target));
      if (fileText(plan.target) !== plan.before) throw new Error('Host settings changed during installation');
      if (plan.before === plan.after) continue;
      replaceText(plan.target, plan.after);
      written.push(plan);
    }
    return plans.map(plan => plan.target);
  } catch (error) {
    for (const plan of written.reverse()) {
      if (fileText(plan.target) !== plan.after) continue;
      if (plan.before === null) unlinkSync(plan.target); else replaceText(plan.target, plan.before);
    }
    throw error;
  }
}

/** nativeLauncher 解析 Windows 的官方 npm launcher，不把参数拼成 shell 命令。
 * Args:
 *   host: claude 或 codex。
 * Returns:
 *   当前 Node 可执行的官方宿主脚本。
 * Errors:
 *   缺少官方 npm 布局时失败，不执行未知 cmd 文件。
 */
async function nativeLauncher(host) {
  // Resolve PATH with Unicode-aware filesystem calls. Windows where.exe can
  // fail on non-ASCII profile paths despite the shim being directly readable.
  const paths = (process.env.PATH ?? process.env.Path ?? '').split(delimiter).filter(Boolean);
  const shim = paths.map(value => join(value.replace(/^"(.*)"$/, '$1'), host + '.cmd')).find(path => {
    try { return isAbsolute(path) && lstatSync(path).isFile(); } catch (error) { if (['ENOENT', 'ENOTDIR'].includes(error.code)) return false; throw error; }
  });
  if (!shim) throw new Error('Native plugin manager launcher unavailable');
  const identity = host === 'codex' ? '@openai/codex' : '@anthropic-ai/claude-code';
  const root = join(dirname(shim), 'node_modules', identity);
  const manifest = JSON.parse(fileText(join(root, 'package.json')) ?? '{}');
  const script = join(root, host === 'codex' ? 'bin/codex.js' : 'cli.js');
  const stat = lstatSync(script);
  if (manifest.name !== identity || !stat.isFile() || stat.isSymbolicLink()) throw new Error('Native plugin manager launcher unavailable');
  return script;
}

/** runNative 调用宿主原生插件生命周期，不启动模型或自行写插件缓存。
 * Args:
 *   host: claude 或 codex；WorkBuddy 没有已验证的注册 CLI。
 *   args: 固定的原生插件命令参数数组。
 * Returns:
 *   原生命令的 stdout 文本。
 * Errors:
 *   宿主缺失、超时或失败时返回不含 stderr 和凭证的异常。
 */
export async function runNative(host, args) {
  if (!['claude', 'codex'].includes(host)) throw new Error('Native plugin manager unavailable');
  const options = { timeout: 180_000, maxBuffer: 1024 * 1024, windowsHide: true, shell: false };
  try {
    if (process.platform !== 'win32') return (await exec(host, args, options)).stdout;
    try { return (await exec(host + '.exe', args, options)).stdout; }
    catch (error) { if (error.code !== 'ENOENT') throw error; }
    const launcher = await nativeLauncher(host);
    return (await exec(process.execPath, [launcher, ...args], options)).stdout;
  } catch { throw new Error('Native plugin manager failed; check the selected host installation and retry'); }
}

/** samePath 规范本地来源路径，包括 Windows 原生工具返回的扩展路径前缀。
 * Args:
 *   actual: 原生工具报告的本地路径。
 *   expected: 固定的全局包路径。
 * Returns:
 *   两个绝对路径是否指向同一词法位置。
 */
function samePath(actual, expected) {
  if (typeof actual !== 'string') return false;
  const clean = value => value.replace(/^\\\\\?\\UNC\\/i, '\\\\').replace(/^\\\\\?\\/, '');
  if (!isAbsolute(clean(actual))) return false;
  const left = resolve(clean(actual)), right = resolve(clean(expected));
  return process.platform === 'win32' ? left.toLowerCase() === right.toLowerCase() : left === right;
}

/** validateNativePayload 确保原生管理器读取同版本包内的受限载荷路径。
 * Args:
 *   root: 稳定的全局 npm 包目录。
 *   host: 所选宿主。
 *   version: 本次已核验的 CLI 版本。
 * Errors:
 *   manifest、catalog 或 hooks 缺失、错版或越界时失败。
 */
function validateNativePayload(root, host, version) {
  const directories = { claude: '.claude-plugin', codex: '.codex-plugin', workbuddy: '.codebuddy-plugin' };
  const manifest = JSON.parse(fileText(join(root, directories[host], 'plugin.json')) ?? '{}');
  if (manifest.name !== 'ccneed' || manifest.version !== version || manifest.skills !== './skills' || manifest.hooks !== './hooks/' + host + '.json') throw new Error('Native plugin payload does not match the CLI release');
  const hooks = JSON.parse(fileText(join(root, 'hooks', host + '.json')) ?? '{}');
  if (!hooks.hooks || typeof hooks.hooks !== 'object' || Array.isArray(hooks.hooks)) throw new Error('Native plugin hooks are missing');
  const catalogDirectory = host === 'codex' ? '.agents/plugins' : directories[host];
  const catalog = JSON.parse(fileText(join(root, catalogDirectory, 'marketplace.json')) ?? '{}');
  const entry = Array.isArray(catalog.plugins) ? catalog.plugins.find(value => value.name === 'ccneed') : null;
  const source = host === 'codex' ? { source: 'local', path: './' } : './';
  if (catalog.name !== 'ccneed' || !entry || !sameConfig(entry.source, source) || (entry.version !== undefined && entry.version !== version)) throw new Error('Native marketplace source is invalid');
}

/** nativeJSON 读取有界原生 JSON 输出，不把人类提示误认为安装成功。
 * Args:
 *   native: 可注入的原生进程边界。
 *   host: 当前宿主。
 *   args: 原生命令参数。
 * Returns:
 *   解析后的原生 JSON 数据。
 * Errors:
 *   输出不是有界 JSON 时失败。
 */
async function nativeJSON(native, host, args) {
  const stdout = await native(host, args);
  if (typeof stdout !== 'string' || stdout.length > 1024 * 1024) throw new Error('Invalid native plugin response');
  return JSON.parse(stdout.replace(/^\uFEFF/, ''));
}

/** ensureMarketplace 仅注册稳定包目录，拒绝接管同名的其他来源。
 * Args:
 *   native: 原生进程边界。
 *   host: claude 或 codex。
 *   root: 稳定全局包目录。
 * Returns:
 *   已核验的原生 marketplace 记录。
 * Errors:
 *   同名异源、输出无效或原生注册失败时停止。
 */
async function ensureMarketplace(native, host, root) {
  const read = async () => {
    const value = await nativeJSON(native, host, ['plugin', 'marketplace', 'list', '--json']);
    const rows = host === 'claude' ? value : value.marketplaces;
    if (!Array.isArray(rows)) throw new Error('Invalid native marketplace response');
    const matches = rows.filter(value => value.name === 'ccneed');
    if (matches.length > 1) throw new Error('Ambiguous native marketplace');
    return matches[0];
  };
  let marketplace = await read();
  if (!marketplace) {
    await native(host, ['plugin', 'marketplace', 'add', root, ...(host === 'claude' ? ['--scope', 'user'] : ['--json'])]);
    marketplace = await read();
  }
  const path = host === 'claude' ? marketplace?.path ?? marketplace?.installLocation : marketplace?.root;
  if (!samePath(path, root) || (host === 'claude' && !['directory', 'local'].includes(marketplace?.source))) throw new Error('Existing CCNeed marketplace has another source');
  return marketplace;
}

/** installedPlugin 读取本插件源的原生安装状态，不查询无关远程源。
 * Args:
 *   native: 原生进程边界。
 *   host: claude 或 codex。
 * Returns:
 *   当前插件记录，未安装时返回 null。
 * Errors:
 *   无效或重复的原生安装记录返回异常。
 */
async function installedPlugin(native, host) {
  const value = await nativeJSON(native, host, ['plugin', 'list', '--json', ...(host === 'codex' ? ['--marketplace', 'ccneed'] : [])]);
  const rows = host === 'claude' ? value : value.installed;
  if (!Array.isArray(rows)) throw new Error('Invalid native plugin response');
  const matches = rows.filter(row => host === 'claude' ? row.id === 'ccneed@ccneed' && row.scope === 'user' : row.pluginId === 'ccneed@ccneed');
  if (matches.length > 1) throw new Error('Ambiguous native plugin installation');
  return matches[0] ?? null;
}

/** installNativePlugin 委托宿主注册与升级，逐宿主报告实际版本和待完成操作。
 * Args:
 *   native: 原生进程边界，接收 host 和参数数组。
 *   host: 当前选择的宿主。
 *   root: 已验证的全局包目录。
 *   version: 期望的同包版本。
 * Returns:
 *   原生安装结果；错误和 WorkBuddy 手动步骤不会伪装为 installed。
 */
export async function installNativePlugin(native, host, root, version) {
  const base = { host, plugin_id: 'ccneed@ccneed', requested_version: version, version: null, marketplace_root: root };
  try {
    validateNativePayload(root, host, version);
    if (host === 'workbuddy') return { ...base, status: 'action_required', next_steps: ['Open WorkBuddy desktop’s native plugin manager and register the local CCNeed marketplace at ' + root, 'Install and enable ccneed@ccneed version ' + version + '; verify that version in the desktop plugin manager', 'Trust the bundled hooks and restart WorkBuddy with default permissions; no supported desktop CLI registration API was used'] };
    await ensureMarketplace(native, host, root);
    let installed = await installedPlugin(native, host);
    if (installed?.enabled === false) return { ...base, status: 'action_required', version: installed.version ?? null, next_steps: ['The native plugin is disabled; review and enable it in the host plugin manager before retrying'] };
    if (!installed || installed.version !== version) {
      if (host === 'claude' && installed) await native(host, ['plugin', 'marketplace', 'update', 'ccneed']);
      await native(host, ['plugin', host === 'claude' ? (installed ? 'update' : 'install') : 'add', 'ccneed@ccneed', ...(host === 'claude' ? ['--scope', 'user'] : ['--json'])]);
      installed = await installedPlugin(native, host);
    }
    const sourceMatches = host === 'claude' || (installed?.installed === true && installed?.source?.source === 'local' && samePath(installed.source.path, root));
    if (!installed || installed.version !== version || installed.enabled !== true || !sourceMatches) return { ...base, status: 'action_required', version: installed?.version ?? null, next_steps: ['The native manager did not verify the requested enabled plugin version; complete installation in the host plugin manager and retry'] };
    return { ...base, status: 'installed', version: installed.version, restart_required: true, hook_trust_required: true };
  } catch { return { ...base, status: 'failed', error: 'native_plugin_registration_failed', next_steps: ['Check the selected host supports native plugin marketplaces and that ccneed is sourced from this exact package directory, then retry; existing custom configuration was preserved'] }; }
}
