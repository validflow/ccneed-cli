import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const workflow = readFileSync(new URL('../../.github/workflows/cx-review.yml', import.meta.url), 'utf8');

test('automatic review stays paused without losing the same-repository guard', () => {
  const review = workflow.split('\n  review:')[1]?.split('\n  post-review:')[0];
  const post = workflow.split('\n  post-review:')[1];
  assert.match(review, /if: false && github\.event\.pull_request\.head\.repo\.full_name == github\.repository/);
  assert.match(post, /needs\.review\.result != 'skipped'/);
});

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
  assert.doesNotMatch(post, /node scripts\/review\/validate\.mjs/);
  assert.match(post, /REVIEW_RESULT/);
  assert.match(post, /gh pr comment "\$PR_NUMBER" --repo "\$GITHUB_REPOSITORY" --body-file review-output\/codex-review\.md/);
  assert.match(post, /test -s review-output\/codex-review\.md/);
  assert.match(post, /test "\$REVIEW_RESULT" = success/);
  assert.match(post, /pull-requests:\s*write/);
  assert.match(post, /actions\/download-artifact@/);
});
