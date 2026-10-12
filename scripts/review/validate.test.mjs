import assert from 'node:assert/strict';
import test from 'node:test';
import { validateReport } from './validate.mjs';

const complete = `## Summary

**Status: Complete.**

## Critical Issues

None.

## Important Issues

None.

## Minor Issues

One harmless observation.

## Checklist Compliance

Four passes and second-pass validation completed.
`;

test('a complete review with no blockers passes the CI gate', () => {
  assert.doesNotThrow(() => validateReport(complete));
});

test('missing or incomplete reviews fail closed', () => {
  assert.throws(() => validateReport(''), /review/i);
  assert.throws(() => validateReport(complete.replace('Complete.', 'Incomplete.')), /complete/i);
  assert.throws(() => validateReport(complete.replace('## Checklist Compliance', '## Other')), /Checklist Compliance/);
  assert.throws(() => validateReport(complete.replace('Four passes and second-pass validation completed.', '')), /Checklist Compliance/);
  assert.throws(() => validateReport(complete.replace('## Summary', '## Summary\n\n**Status: Incomplete.**')), /complete/i);
  assert.throws(() => validateReport(complete.replace('**Status: Complete.**', 'Review can eventually be **Status: Complete.**')), /complete/i);
});

test('a confirmed Critical or Important issue fails the CI gate', () => {
  assert.throws(() => validateReport(complete.replace('## Critical Issues\n\nNone.', '## Critical Issues\n\n- Credential leak.')), /Critical/);
  assert.throws(() => validateReport(complete.replace('## Important Issues\n\nNone.', '## Important Issues\n\n- Broken contract.')), /Important/);
});
