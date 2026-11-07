import { lstatSync, readFileSync, mkdirSync, writeFileSync, renameSync, unlinkSync } from 'node:fs';
import { dirname, isAbsolute, join, resolve } from 'node:path';
import { randomUUID } from 'node:crypto';

const hostDirs = { codex: '.codex', claude: '.claude', workbuddy: '.workbuddy' };
const commands = ['ccneed', 'ccneed.cmd', 'ccneed.exe'];

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

/** hookHandler 生成当前平台可安全引用绝对路径的宿主命令。
 * Args:
 *   root: 固定全局 npm 包目录。
 *   host: 明确选择的宿主。
 * Returns:
 *   不经动态 shell 插值的命令配置。
 * Errors:
 *   路径包含换行或 NUL 时拒绝生成配置。
 */
function hookHandler(root, host) {
  const args = [process.execPath, join(root, 'skills', 'ccneed', 'scripts', 'acquisition.mjs'), '--host', host];
  if (args.some(value => /[\r\n\0]/.test(value))) throw new Error('Invalid hook path');
  const posix = args.map(value => "'" + value.replaceAll("'", "'\\''") + "'").join(' ');
  const powershell = '& ' + args.map(value => "'" + value.replaceAll("'", "''") + "'").join(' ');
  if (host === 'codex') return { type: 'command', command: posix, commandWindows: powershell, timeout: 10 };
  return { type: 'command', command: process.platform === 'win32' ? powershell : posix, ...(process.platform === 'win32' ? { shell: 'powershell' } : {}), timeout: 10 };
}

/** mergeSettings 保留用户设置，仅添加采集展示与执行保护。
 * Args:
 *   before: 已有配置原文或 null。
 *   root: 固定全局 npm 包目录。
 *   host: 明确选择的宿主。
 * Returns:
 *   合并后的 JSON 配置原文。
 * Errors:
 *   配置格式无效或发现其他来源的同名 hook 时停止。
 */
function mergeSettings(before, root, host) {
  const settings = before === null ? {} : JSON.parse(before.replace(/^\uFEFF/, ''));
  if (!settings || Array.isArray(settings) || typeof settings !== 'object') throw new Error('Plugin settings must be a JSON object');
  const handler = hookHandler(root, host);
  settings.hooks ??= {};
  if (!settings.hooks || Array.isArray(settings.hooks) || typeof settings.hooks !== 'object') throw new Error('Plugin hooks must be a JSON object');
  for (const event of ['SessionStart', 'PreToolUse', 'PostToolUse']) {
    const group = { ...(event === 'SessionStart' ? {} : { matcher: 'Bash|PowerShell|exec_command' }), hooks: [handler] };
    const entries = settings.hooks[event] ?? [];
    if (!Array.isArray(entries)) throw new Error('Plugin event hooks must be an array');
    const desired = JSON.stringify(group);
    const owned = entries.filter(entry => JSON.stringify(entry).includes('acquisition.mjs'));
    if (owned.some(entry => JSON.stringify(entry) !== desired)) throw new Error('Existing custom CCNeed hook will not be overwritten');
    settings.hooks[event] = entries.some(entry => JSON.stringify(entry) === desired) ? entries : [...entries, group];
  }
  if (host !== 'codex') {
    settings.permissions ??= {};
    if (!settings.permissions || Array.isArray(settings.permissions) || typeof settings.permissions !== 'object') throw new Error('Plugin permissions must be a JSON object');
    const ask = settings.permissions.ask ?? [];
    if (!Array.isArray(ask) || ask.some(value => typeof value !== 'string')) throw new Error('Plugin ask rules must be strings');
    const added = commands.flatMap(binary => ['Bash', 'PowerShell'].flatMap(tool => [`${tool}(${binary} kb start *)`, `${tool}(${binary} tasks reply *)`]));
    settings.permissions.ask = [...new Set([...ask, ...added])];
  }
  if (host === 'workbuddy') {
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
  return JSON.stringify(settings, null, 2) + '\n';
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
 *   每个配置目标的原文和计划内容。
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
    const files = [{ home, host, target, before, after: mergeSettings(before, root, host), root }];
    if (host === 'codex') {
      const target = join(directory, 'rules', 'ccneed.rules');
      checkParents(home, dirname(target));
      const before = fileText(target);
      const after = codexRules();
      if (before !== null && before !== after) throw new Error('Existing custom CCNeed rules will not be overwritten');
      files.push({ home, host, target, before, after, root });
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
