import assert from 'node:assert/strict';
import test from 'node:test';
import { admit, consolidate, render } from './gate.mjs';

const sha = 'a'.repeat(40), base = 'b'.repeat(40);
const pr = { number: 7, state: 'open', draft: false, author_association: 'MEMBER',
  head: { sha, repo: { full_name: 'validflow/ccneed-cli' } },
  base: { sha: base, ref: 'main', repo: { full_name: 'validflow/ccneed-cli' } } };
const context = { repository: 'validflow/ccneed-cli', event: 'pull_request_target',
  expectedHead: sha, authorPermission: 'write', actorPermission: 'write' };

test('internal trusted author admits the exact current head and base', () => {
  assert.deepEqual(admit(pr, context), { allowed: true, head: sha, base, number: 7 });
});
test('fork never automatically enters the credential-bearing runner', () => {
  const fork = { ...pr, head: { sha, repo: { full_name: 'outside/fork' } } };
  assert.equal(admit(fork, context).allowed, false);
  assert.equal(admit(fork, { ...context, event: 'workflow_dispatch', approvedHead: sha }).allowed, true);
});
test('manual approval binds a maintainer and an immutable SHA', () => {
  for (const change of [{ approvedHead: 'c'.repeat(40) }, { actorPermission: 'read' }, { approvedHead: '' }]) {
    assert.throws(() => admit(pr, { ...context, event: 'workflow_dispatch', approvedHead: sha, ...change }));
  }
});
test('stale event, unknown author, deleted repo and wrong base fail closed', () => {
  assert.throws(() => admit(pr, { ...context, expectedHead: 'c'.repeat(40) }));
  assert.equal(admit(pr, { ...context, authorPermission: 'read' }).allowed, false);
  assert.equal(admit({ ...pr, author_association: 'NONE' }, context).allowed, false);
  assert.throws(() => admit({ ...pr, head: { sha, repo: null } }, context));
  assert.throws(() => admit({ ...pr, base: { ...pr.base, ref: 'other' } }, context));
});
const roles = ['bug-detector', 'arch-compliance', 'security-audit', 'logic-validator'];
const clean = () => roles.map(role => ({ role, complete: true, findings: [] }));
const finding = { id: 'auth-1', severity: 'important', path: 'internal/auth.go', line: 8,
  title: 'Credential leak', explanation: 'A definite introduced flaw.', evidence: 'New header path.' };
test('all four completed independent passes produce a clean report', () => {
  const result = consolidate(clean(), [], ['internal/auth.go']);
  assert.equal(result.state, 'success');
  assert.match(render(result, pr), /Critical Issues[\s\S]*Important Issues[\s\S]*Minor Issues[\s\S]*Checklist Compliance/);
});
test('missing, duplicate, incomplete and arbitrary markdown cannot pass', () => {
  for (const passes of [clean().slice(1), [...clean(), clean()[0]],
    clean().map((pass, i) => i ? pass : { ...pass, complete: false }), 'No issues']) {
    assert.throws(() => consolidate(passes, [], []));
  }
});
test('critical and important require explicit independent verification', () => {
  const passes = clean(); passes[2].findings = [finding];
  assert.throws(() => consolidate(passes, [], ['internal/auth.go']));
  const confirmed = consolidate(passes, [{ id: finding.id, verdict: 'confirmed', rationale: 'Validated against changed code.' }], ['internal/auth.go']);
  assert.equal(confirmed.state, 'failure');
  assert.equal(confirmed.findings.length, 1);
  const rejected = consolidate(passes, [{ id: finding.id, verdict: 'false_positive', rationale: 'Existing guard handles it.' }], ['internal/auth.go']);
  assert.equal(rejected.state, 'success');
  assert.throws(() => consolidate(passes, [{ id: finding.id, verdict: 'confirmed', rationale: '' }], ['internal/auth.go']));
});
test('invalid locations, severities and duplicate finding IDs are rejected', () => {
  for (const change of [{ severity: 'clean' }, { path: '../secret' }, { line: 0 }, { evidence: '' }]) {
    const passes = clean(); passes[0].findings = [{ ...finding, ...change }];
    assert.throws(() => consolidate(passes, [], ['internal/auth.go']));
  }
  const passes = clean(); passes[0].findings = [{ ...finding, severity: 'minor' }];
  passes[1].findings = [{ ...finding, severity: 'minor' }];
  assert.throws(() => consolidate(passes, [], ['internal/auth.go']));
});
test('minor is advisory; report text cannot activate mentions or HTML', () => {
  const passes = clean(); passes[0].findings = [{ ...finding, severity: 'minor', title: '<img src=x> @all [link](https://evil)' }];
  const result = consolidate(passes, [], ['internal/auth.go']);
  assert.equal(result.state, 'success');
  const report = render(result, pr);
  assert.ok(!report.includes('<img')); assert.ok(!report.includes('@all'));
  assert.ok(!report.includes('[link](https://evil)'));
});
