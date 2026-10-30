import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';

const workflow = readFileSync(new URL('../../.github/workflows/release.yml', import.meta.url), 'utf8');

test('release checks and builds cannot request npm publishing identity', () => {
  const build = workflow.split('\n  build:')[1]?.split('\n  publish:')[0];
  const publish = workflow.split('\n  publish:')[1];
  assert.ok(build, 'release workflow needs a separate build job');
  assert.ok(publish, 'release workflow needs a publishing job');
  assert.doesNotMatch(build, /id-token:\s*write/);
  assert.match(build, /node scripts\/release\/build\.mjs/);
  assert.match(build, /bash scripts\/check\.sh/);
  assert.match(publish, /id-token:\s*write/);
  assert.match(publish, /actions\/download-artifact@/);
  assert.doesNotMatch(build, /node scripts\/release\/tag\.mjs|node scripts\/release\/publish\.mjs/);
  assert.doesNotMatch(publish, /node scripts\/release\/tag\.mjs/);
  assert.match(publish, /node scripts\/release\/publish\.mjs/);
  assert.doesNotMatch(publish, /node scripts\/release\/build\.mjs|bash scripts\/check\.sh|go test -race/);
});

test('a failed unchanged version can only be reissued explicitly from main', () => {
  assert.match(workflow, /workflow_dispatch:/);
  assert.match(workflow, /retry_version:/);
  assert.match(workflow, /github\.ref == 'refs\/heads\/main'/);
  assert.match(workflow, /RETRY_VERSION: \$\{\{ inputs\.retry_version \}\}/);
});

test('installer dependencies are installed without lifecycle scripts before tests, builds and publication', () => {
  const quality = readFileSync(new URL('../../.github/workflows/quality.yml', import.meta.url), 'utf8');
  const jobs = [
    [quality.split('\n  checks:')[1]?.split('\n  quality:')[0], 'npm test'],
    [workflow.split('\n  build:')[1]?.split('\n  publish:')[0], 'npm test'],
    [workflow.split('\n  publish:')[1]?.split('\n  notify:')[0], 'node scripts/release/publish.mjs'],
  ];
  for (const [job, command] of jobs) {
    assert.ok(job, 'expected workflow job must exist');
    const install = job.indexOf('npm ci --ignore-scripts');
    assert.ok(install >= 0, 'jobs importing installer modules must install locked dependencies without running postinstall');
    assert.ok(install < job.indexOf(command), 'dependencies must be available before importing installer modules');
  }
});
