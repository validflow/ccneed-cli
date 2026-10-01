import assert from 'node:assert/strict';
import test from 'node:test';
import path from 'node:path';
import os from 'node:os';
import { mkdtemp, writeFile, rm } from 'node:fs/promises';
import { bundle, review, modelEnvironment, invokeCodex, runnerOptions } from './pipeline.mjs';
const head = 'a'.repeat(40), base = 'b'.repeat(40);
const prepared = { head, base, number: 3, allowed: true, metadata: { title: 'Improve CLI', body: '' },
  changed: [{ filename: 'a.go', status: 'modified', additions: 1, deletions: 1, patch: '@@ -1 +1 @@\n-old\n+new' }] };
test('bundle contains full patch and trusted base content without executing changed code', async () => {
  const calls = [];
  const result = await bundle(prepared, async path => { calls.push(path); return 'trusted old'; });
  assert.equal(result.files[0], 'a.go');
  assert.ok(result.context.includes('trusted old'));
  assert.ok(result.context.includes('+new'));
  assert.deepEqual(calls, ['a.go']);
});
test('missing/truncated/binary patches and unsafe filenames fail closed', async () => {
  for (const change of [{ patch: undefined }, { additions: 2 }, { filename: '../auth.json' }, { filename: 'C:\\auth.json' }]) {
    await assert.rejects(bundle({ ...prepared, changed: [{ ...prepared.changed[0], ...change }] }, async () => 'old'));
  }
});
test('oversized context is rejected instead of silently truncating', async () => {
  await assert.rejects(bundle(prepared, async () => 'x'.repeat(800000)));
});
test('four independent calls, followed by a separate verification call for blockers', async () => {
  const calls = [];
  const input = await bundle(prepared, async () => 'old');
  const result = await review(input, async (role, prompt) => {
    calls.push({ role, prompt });
    if (role === 'verify') return { decisions: [{ id: 'security-audit-1', verdict: 'confirmed', rationale: 'Independent code validation.' }] };
    return { role, complete: true, findings: role === 'security-audit' ? [{ id: 'security-audit-1', severity: 'important', path: 'a.go', line: 1,
      title: 'Bug', explanation: 'Introduced behavior is incorrect.', evidence: 'Changed line.' }] : [] };
  });
  assert.deepEqual(calls.map(call => call.role).sort(), ['arch-compliance', 'bug-detector', 'logic-validator', 'security-audit', 'verify']);
  assert.equal(result.passes.length, 4);
  assert.equal(result.verification[0].verdict, 'confirmed');
  assert.ok(calls.every(call => call.prompt.includes('untrusted')));
});
test('a missing independent pass is a failing pipeline, not a clean aggregate', async () => {
  const input = await bundle(prepared, async () => 'old');
  await assert.rejects(review(input, async role => role === 'bug-detector' ? null : { role, complete: true, findings: [] }));
});
test('model environment contains no inherited GitHub token, user config, hooks or arbitrary secrets', () => {
  const env = modelEnvironment({ PATH: '/bin', GH_TOKEN: 'write-secret', GITHUB_TOKEN: 'secret', AWS_SECRET_ACCESS_KEY: 'aws', HOME: '/host-home' }, '/isolated');
  assert.deepEqual(env, { PATH: '/bin', HOME: '/isolated', CODEX_HOME: path.join('/isolated', 'auth'), CI: 'true' });
});
test('empty workflow variables still resolve safe configured defaults', () => {
  assert.deepEqual(runnerOptions({ HOME: '/runner', CX_AUTH_FILE: '', CX_CODEX_BINARY: '' }), {
    authFile: path.join('/runner', '.codex', 'auth.json'), executable: 'codex' });
  assert.equal(runnerOptions({ HOME: '/runner', CX_CODEX_BINARY: '/trusted/codex', CX_AUTH_FILE: '/trusted/auth' }).executable, '/trusted/codex');
});
test('actual subprocess invocation is tool-free and receives only isolated process inputs', async () => {
  const isolated = await mkdtemp(path.join(os.tmpdir(), 'ccneed-cx-test-'));
  try {
    const fake = path.join(isolated, 'fake-codex.mjs');
    await writeFile(fake, `import fs from 'node:fs';
      const args = process.argv.slice(2); let input = '';
      for await (const chunk of process.stdin) input += chunk;
      fs.writeFileSync(args[args.indexOf('-o') + 1], JSON.stringify({ args, input, environment: process.env, cwd: process.cwd() }));`);
    const output = await invokeCodex('bug-detector', 'untrusted patch input', isolated, process.execPath, [fake]);
    assert.equal(output.input, 'untrusted patch input');
    assert.ok(output.args.includes('--ignore-user-config'));
    assert.ok(output.args.includes('--ignore-rules'));
    assert.ok(output.args.includes('--strict-config'));
    assert.equal(output.args[output.args.indexOf('--sandbox') + 1], 'read-only');
    assert.equal(output.args[output.args.indexOf('--disable') + 1], 'shell_tool');
    assert.ok(output.args.includes('web_search="disabled"'));
    assert.ok(output.args.includes('features.multi_agent=false'));
    assert.equal(output.environment.HOME, isolated);
    assert.equal(output.environment.CODEX_HOME, path.join(isolated, 'auth'));
    // Windows may inject mandatory OS profile variables into CreateProcess. The production
    // review pool is macOS; publishing/API secrets must never be inherited on either platform.
    for (const key of ['GH_TOKEN', 'GITHUB_TOKEN', 'AWS_SECRET_ACCESS_KEY', 'OPENAI_API_KEY', 'CODEX_ACCESS_TOKEN']) {
      assert.equal(output.environment[key], undefined, key);
    }
    if (process.platform !== 'win32') assert.deepEqual(Object.keys(output.environment).sort(), ['CI', 'CODEX_HOME', 'HOME', 'PATH']);
    assert.equal(output.cwd, isolated);
    assert.ok(!output.args.some(value => value.includes('dangerously')));
  } finally { await rm(isolated, { recursive: true, force: true }); }
});
