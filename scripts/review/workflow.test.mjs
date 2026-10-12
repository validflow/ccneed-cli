import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const workflow = readFileSync(new URL('../../.github/workflows/cx-review.yml', import.meta.url), 'utf8');

test('Codex review is read-only and posting is isolated in a separate job', () => {
  const review = workflow.split('\n  review:')[1]?.split('\n  post-review:')[0];
  const post = workflow.split('\n  post-review:')[1];
  const codex = review?.split('      - name: Run Codex CLI review')[1]?.split('      - name: Preserve review report')[0];
  assert.ok(review, 'review job is required');
  assert.ok(post, 'posting job is required');
  assert.match(review, /pull-requests:\s*read/);
  assert.doesNotMatch(review, /pull-requests:\s*write|issues:\s*write/);
  assert.ok(codex, 'Codex invocation is required');
  assert.doesNotMatch(codex, /GH_TOKEN:|GITHUB_TOKEN:/);
  assert.match(review, /persist-credentials:\s*false/);
  assert.doesNotMatch(review, /node scripts\/review\/validate\.mjs/);
  assert.doesNotMatch(review, /ref: refs\/pull\//);
  assert.match(review, /ref: \$\{\{ github\.event\.pull_request\.base\.sha \}\}/);
  assert.match(review, /gh pr diff/);
  assert.match(workflow, /pull_request_target:/);
  assert.match(post, /ref: \$\{\{ github\.event\.pull_request\.base\.sha \}\}/);
  assert.match(post, /node scripts\/review\/validate\.mjs/);
  assert.match(post, /REVIEW_RESULT/);
  assert.ok(post.indexOf('Validate trusted review result') < post.indexOf('Post report or failure notice'));
  assert.match(post, /steps\.report-validation\.outcome/);
  assert.match(post, /test "\$VALIDATION_OUTCOME" = success/);
  assert.match(post, /pull-requests:\s*write/);
  assert.match(post, /actions\/download-artifact@/);
});
