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
  assert.match(publish, /node scripts\/release\/tag\.mjs/);
  assert.doesNotMatch(publish, /node scripts\/release\/build\.mjs|bash scripts\/check\.sh|go test -race/);
});
