// This module has no network or subprocess capabilities. CI owns all side effects.
export const roles = ['bug-detector', 'arch-compliance', 'security-audit', 'logic-validator'];
const shaPattern = /^[a-f0-9]{40}$/;
const writers = new Set(['write', 'maintain', 'admin']);

// admit binds authorization to the live PR identity and immutable revision.
// Args: pr is live GitHub metadata; context contains trusted event/SHA/permission facts.
// Returns: authorization, immutable head/base and PR number.
// Errors: rejects stale, closed, draft, malformed or unauthorized manual events.
export function admit(pr, context) {
  if (pr.state !== 'open' || pr.draft || !Number.isSafeInteger(pr.number) || pr.number < 1) throw new Error('PR is not reviewable');
  if (pr.base?.repo?.full_name !== context.repository || pr.base.ref !== 'main') throw new Error('Untrusted base');
  if (!pr.head?.repo || !shaPattern.test(pr.head.sha) || !shaPattern.test(pr.base.sha)) throw new Error('Invalid revision');
  let allowed;
  if (context.event === 'workflow_dispatch') {
    if (!writers.has(context.actorPermission) || context.approvedHead !== pr.head.sha) throw new Error('Approval is unauthorized or stale');
    allowed = true;
  } else if (context.event === 'pull_request_target') {
    if (context.expectedHead !== pr.head.sha) throw new Error('Event is stale');
    allowed = pr.head.repo.full_name === context.repository && writers.has(context.authorPermission)
      && ['OWNER', 'MEMBER', 'COLLABORATOR'].includes(pr.author_association);
  } else {
    throw new Error('Unsupported event');
  }
  return { allowed, head: pr.head.sha, base: pr.base.sha, number: pr.number };
}

// requiredText rejects empty, non-string and over-limit model text.
// Args: value is untrusted output; limit is the maximum JavaScript string length.
// Returns: whether the value is usable non-empty text.
function requiredText(value, limit = 8000) {
  return typeof value === 'string' && value.trim().length > 0 && value.length <= limit;
}

// validatePass rejects incomplete model output rather than trusting a "clean" string.
// Args: pass is one output; changedFiles is the allowlist; ids tracks cross-pass uniqueness.
// Returns: nothing; validated finding IDs are added to ids.
// Errors: rejects incomplete passes or invalid finding identity/location/evidence.
function validatePass(pass, changedFiles, ids) {
  if (!roles.includes(pass?.role) || pass.complete !== true || !Array.isArray(pass.findings)) throw new Error('Incomplete pass');
  if (pass.findings.length > 100) throw new Error('Too many findings');
  for (const finding of pass.findings) {
    if (!requiredText(finding.id, 120) || ids.has(finding.id)) throw new Error('Duplicate or invalid finding ID');
    if (!['critical', 'important', 'minor'].includes(finding.severity)) throw new Error('Invalid severity');
    if (!changedFiles.includes(finding.path) || finding.path.split('/').includes('..')) throw new Error('Invalid finding path');
    if (!Number.isSafeInteger(finding.line) || finding.line < 1) throw new Error('Invalid line');
    for (const field of ['title', 'explanation', 'evidence']) if (!requiredText(finding[field])) throw new Error('Incomplete finding');
    ids.add(finding.id);
  }
}

// consolidate requires four passes and complete second-pass disposition of blockers.
// Args: passes are four raw outputs; verification is independent; changedFiles limits scope.
// Returns: safe status, retained findings and validation statistics.
// Errors: rejects missing/incomplete/duplicate passes or verification dispositions.
export function consolidate(passes, verification, changedFiles) {
  if (!Array.isArray(passes) || passes.length !== roles.length || new Set(passes.map(pass => pass.role)).size !== roles.length) throw new Error('Four independent passes required');
  const ids = new Set();
  for (const pass of passes) validatePass(pass, changedFiles, ids);
  const findings = passes.flatMap(pass => pass.findings.map(finding => ({ ...finding, role: pass.role })));
  const blockers = findings.filter(finding => finding.severity !== 'minor');
  if (!Array.isArray(verification) || verification.length !== blockers.length) throw new Error('Second-pass verification required');
  const decisions = new Map();
  for (const decision of verification) {
    if (!blockers.some(finding => finding.id === decision.id) || decisions.has(decision.id)
      || !['confirmed', 'false_positive'].includes(decision.verdict) || !requiredText(decision.rationale)) throw new Error('Invalid verification');
    decisions.set(decision.id, decision);
  }
  const kept = findings.filter(finding => finding.severity === 'minor' || decisions.get(finding.id)?.verdict === 'confirmed');
  return { state: kept.some(finding => finding.severity !== 'minor') ? 'failure' : 'success', findings: kept,
    passes: roles, verified: blockers.length, discarded: blockers.length - kept.filter(finding => finding.severity !== 'minor').length };
}

// safeText neutralizes active Markdown, HTML and mentions in model-generated prose.
// Args: value is text to publish as inert prose.
// Returns: escaped prose without executable HTML, links or mentions.
function safeText(value) {
  return String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')
    .replaceAll('@', '@\u200b').replace(/[\\`*_\[\]()#!|]/g, character => `\\${character}`);
}

// render only emits review text from a validated result, not arbitrary model Markdown.
// Args: result is validated review data; pr supplies bound numeric/revision identity.
// Returns: Markdown with fixed review headings and inert finding text.
export function render(result, pr) {
  const sections = [['critical', 'Critical Issues'], ['important', 'Important Issues'], ['minor', 'Minor Issues']];
  let report = `## CX Review\n\nPR #${pr.number} · head ${pr.head.sha} · base ${pr.base.sha}\n\n### Summary\n\nFour independent passes completed; ${result.verified} blocker candidates verified; ${result.discarded} false positives excluded.\n\n`;
  for (const [severity, title] of sections) {
    report += `### ${title}\n\n`;
    const items = result.findings.filter(finding => finding.severity === severity);
    report += items.length ? items.map(finding => `- ${safeText(finding.path)}:${finding.line} — ${safeText(finding.title)} (${finding.role})\n  ${safeText(finding.explanation)}\n  Evidence: ${safeText(finding.evidence)}`).join('\n') + '\n\n' : 'None.\n\n';
  }
  report += `### Checklist Compliance\n\nAll four review passes completed. ${result.state === 'success' ? 'No confirmed Critical/Important blockers.' : 'Confirmed blockers must be resolved before merge.'}\n`;
  return report;
}
