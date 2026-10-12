import { readFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';

const sections = ['Summary', 'Critical Issues', 'Important Issues', 'Minor Issues', 'Checklist Compliance'];

/** validateReport 校验审查完整性，并拒绝仍有合并阻断项的报告。
 *
 * Args:
 *   report: Codex 审查产生的最终 Markdown 文本。
 *
 * Returns:
 *   校验通过时不返回值。
 *
 * Errors:
 *   报告缺失、不完整、格式不符或包含阻断项时抛出错误。
 */
export function validateReport(report) {
  if (!report?.trim()) throw new Error('Missing review report');
  const positions = sections.map((section) => {
    const match = new RegExp('^## ' + section + '\\s*$', 'm').exec(report);
    if (!match) throw new Error('Missing ' + section + ' section');
    return match.index;
  });
  if (positions.some((position, index) => index > 0 && position <= positions[index - 1])) {
    throw new Error('Review sections are out of order');
  }
  const summary = report.slice(positions[0], positions[1]);
  if (!/^\*\*Status: Complete\.\*\*$/m.test(summary) || /\*\*Status: Incomplete\.\*\*/.test(report)) {
    throw new Error('Review did not complete');
  }
  for (const section of ['Critical Issues', 'Important Issues']) {
    const index = sections.indexOf(section);
    const body = report.slice(positions[index], positions[index + 1]).replace(/^## [^\n]+/, '').trim();
    if (body !== 'None.') throw new Error(section + ' need resolution before merge');
  }
  if (!report.slice(positions[4]).replace(/^## [^\n]+/, '').trim()) {
    throw new Error('Checklist Compliance is empty');
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    validateReport(readFileSync(process.argv[2], 'utf8'));
    console.log('Complete review with no Critical or Important findings');
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
