import { lstatSync, mkdirSync, readFileSync, readlinkSync, readdirSync, symlinkSync, unlinkSync } from 'node:fs';
import { isAbsolute, join, resolve } from 'node:path';
import { createHash } from 'node:crypto';

const hosts = { codex: '.agents', workbuddy: '.codebuddy', claude: '.claude' };

/** fingerprintSkill 为完整技能的文件名和字节生成稳定摘要，不跟随内部链接。
 * Args:
 *   source: 已装配的完整技能目录。
 * Returns:
 *   与平台无关的 SHA-256。
 * Errors:
 *   技能缺失、特殊文件或文件读取失败返回异常。
 */
export function fingerprintSkill(source) {
  validateSkill(source);
  const hash = createHash('sha256');
  hashSkillFiles(source, '', hash);
  return hash.digest('hex');
}

/** hashSkillFiles 按稳定的相对路径顺序计算所有技能文件摘要。
 * Args:
 *   root: 技能源目录。
 *   relative: 使用正斜杠的包内相对路径。
 *   hash: 本次独立摘要对象。
 * Errors:
 *   文件读取错误向上传递。
 */
function hashSkillFiles(root, relative, hash) {
  const path = join(root, relative);
  if (lstatSync(path).isDirectory()) {
    for (const name of readdirSync(path).sort()) hashSkillFiles(root, relative ? relative + '/' + name : name, hash);
  } else {
    const bytes = readFileSync(path);
    hash.update(JSON.stringify([relative, bytes.length]));
    hash.update(bytes);
  }
}

/** validateBundledSkill 校验已发行的技能摘要，防止漏文件、错版和覆盖用户修改。
 * Args:
 *   root: 带发行校验和的 npm 包路径。
 * Errors:
 *   版本、摘要或完整性不匹配时返回异常，要求先保留自定义修改。
 */
export function validateBundledSkill(root) {
  const manifest = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));
  const checksums = JSON.parse(readFileSync(join(root, 'checksums.json'), 'utf8'));
  if (checksums.version !== manifest.version || !/^[a-f0-9]{64}$/.test(checksums.skill_sha256 ?? '') || fingerprintSkill(join(root, 'skills', 'ccneed')) !== checksums.skill_sha256) {
    throw new Error('Skill checksum mismatch: preserve modified Skill before retrying the installer');
  }
}

/** parseAgents 限定用户明确选中的宿主，不扫描或修改其他宿主。
 * Args:
 *   value: 逗号分隔的 codex、workbuddy 或 claude。
 * Returns:
 *   去重后的宿主名称。
 * Errors:
 *   空值、未知名称或重复分隔返回异常。
 */
export function parseAgents(value) {
  if (typeof value !== 'string' || !value.trim()) throw new Error('Choose a host with --agent codex,workbuddy,claude');
  const agents = value.split(',').map(value => value.trim());
  if (agents.some(agent => !Object.hasOwn(hosts, agent))) throw new Error('Unsupported agent; use codex, workbuddy or claude');
  return [...new Set(agents)];
}

/** statOrMissing 查询目标本身，保留悬空链接而不将其误认为不存在。
 * Args:
 *   path: 明确的文件系统目标。
 * Returns:
 *   lstat 结果或 null。
 * Errors:
 *   除不存在外的文件系统错误向上传递。
 */
function statOrMissing(path) {
  try { return lstatSync(path); } catch (error) { if (error.code === 'ENOENT') return null; throw error; }
}

/** sameLink 仅认可指向当前全局 npm 包技能目录的链接。
 * Args:
 *   path: 宿主技能入口。
 *   source: 稳定的全局包技能路径。
 * Returns:
 *   入口是否已经受当前安装管理。
 * Errors:
 *   链接无法读取时向上传递。
 */
function sameLink(path, source) {
  if (!statOrMissing(path)?.isSymbolicLink()) return false;
  const actual = resolve(path, '..', readlinkSync(path));
  return process.platform === 'win32' ? actual.toLowerCase() === source.toLowerCase() : actual === source;
}

/** checkParents 禁止通过宿主父目录链接向另一个位置写入。
 * Args:
 *   parents: 从用户目录到宿主 skills 的明确路径。
 * Errors:
 *   已有链接或非目录返回异常。
 */
function checkParents(parents) {
  for (const path of parents) {
    const stat = statOrMissing(path);
    if (stat && (!stat.isDirectory() || stat.isSymbolicLink())) throw new Error('Skill parent must be a plain directory: ' + path);
  }
}

/** prepareSkillLinks 在修改全局包之前检查所有宿主冲突，不写文件。
 * Args:
 *   home: 当前用户目录；测试可注入隔离目录。
 *   agents: 用户明确选择的宿主名称。
 *   root: npm 返回的稳定全局包目录。
 * Returns:
 *   各宿主入口和同一包内的技能来源。
 * Errors:
 *   非法目录、未知宿主或已有自定义技能返回异常。
 */
export function prepareSkillLinks(home, agents, root) {
  if (!isAbsolute(home) || !isAbsolute(root)) throw new Error('Installation directories must be absolute');
  const source = join(resolve(root), 'skills', 'ccneed');
  return agents.map(agent => {
    if (!Object.hasOwn(hosts, agent)) throw new Error('Unsupported agent');
    const parents = [resolve(home), join(home, hosts[agent]), join(home, hosts[agent], 'skills')];
    checkParents(parents);
    const target = join(parents[2], 'ccneed');
    if (statOrMissing(target) && !sameLink(target, source)) throw new Error('Existing custom Skill will not be overwritten: ' + target);
    return { agent, parents, target, source };
  });
}

/** validateSkill 确认随包技能完整且不含越界链接，不读取任何凭证。
 * Args:
 *   source: 包内完整技能目录。
 * Errors:
 *   缺失必要说明、链接、特殊文件或空文件返回异常。
 */
export function validateSkill(source) {
  const required = ['SKILL.md', 'references/auth.md', 'references/account.md', 'references/tasks.md', 'references/resources.md'];
  inspectSkillTree(source);
  for (const file of required) if (!readFileSync(join(source, file), 'utf8').trim()) throw new Error('Incomplete bundled Skill');
}

/** inspectSkillTree 拒绝技能内部链接和特殊文件。
 * Args:
 *   path: 技能目录树中的一个路径。
 * Errors:
 *   目录树包含链接或特殊文件时返回异常。
 */
function inspectSkillTree(path) {
  const stat = lstatSync(path);
  if (stat.isSymbolicLink()) throw new Error('Bundled Skill cannot contain links');
  if (stat.isDirectory()) {
    for (const file of readdirSync(path)) inspectSkillTree(join(path, file));
  } else if (!stat.isFile()) throw new Error('Bundled Skill must contain only regular files');
}

/** attachSkillLinks 将宿主入口连接到已验证的同版本全局包，不复制第二份技能。
 * Args:
 *   plans: 预检后的用户级技能入口。
 *   createLink: 真实链接创建边界；测试可注入一次文件系统故障。
 * Returns:
 *   已连接的宿主技能路径。
 * Errors:
 *   冲突或链接失败时只撤回本次新增且仍属于本次安装的链接。
 */
export function attachSkillLinks(plans, createLink = symlinkSync) {
  for (const plan of plans) validateSkill(plan.source);
  const created = [];
  try {
    for (const plan of plans) {
      checkParents(plan.parents);
      for (const parent of plan.parents.slice(1)) {
        if (!statOrMissing(parent)) mkdirSync(parent);
        checkParents(plan.parents);
      }
      if (statOrMissing(plan.target)) {
        if (!sameLink(plan.target, plan.source)) throw new Error('Existing custom Skill will not be overwritten: ' + plan.target);
        continue;
      }
      try {
        createLink(plan.source, plan.target, process.platform === 'win32' ? 'junction' : 'dir');
        created.push(plan);
      } catch (error) {
        if (error.code !== 'EEXIST' || !sameLink(plan.target, plan.source)) throw error;
      }
      if (!sameLink(plan.target, plan.source)) throw new Error('Skill link verification failed');
    }
    return plans.map(plan => plan.target);
  } catch (error) {
    for (const plan of created.reverse()) if (sameLink(plan.target, plan.source)) unlinkSync(plan.target);
    throw error;
  }
}
