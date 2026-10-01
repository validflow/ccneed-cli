import assert from 'node:assert/strict';
import test from 'node:test';
import { publish, prepare } from './github.mjs';
const head = 'a'.repeat(40), base = 'b'.repeat(40);
const pr = { number: 9, state: 'open', draft: false, author_association: 'MEMBER',
  user: { login: 'member' }, head: { sha: head, repo: { full_name: 'validflow/ccneed-cli' } },
  base: { sha: base, ref: 'main', repo: { full_name: 'validflow/ccneed-cli' } } };
const roles = ['bug-detector', 'arch-compliance', 'security-audit', 'logic-validator'];
const artifact = { head, base, number: 9, files: ['a.go'], passes: roles.map(role => ({ role, complete: true, findings: [] })), verification: [] };

test('publishing rechecks both live head and base before any write', async () => {
  for (const changed of [{ head: { ...pr.head, sha: 'c'.repeat(40) } }, { base: { ...pr.base, sha: 'c'.repeat(40) } }]) {
    const calls = [];
    const api = async (path, method = 'GET') => { calls.push({ path, method }); return { ...pr, ...changed }; };
    await assert.rejects(publish(api, artifact, 'url', 'success'));
    assert.deepEqual(calls.map(call => call.method), ['GET']);
  }
});
test('head moving during comment publication prevents a final passing status', async () => {
  const calls = []; let reads = 0;
  const api = async (path, method = 'GET', body) => {
    calls.push({ path, method, body });
    if (method === 'GET') return ++reads === 1 ? pr : { ...pr, head: { ...pr.head, sha: 'c'.repeat(40) } };
    return {};
  };
  await assert.rejects(publish(api, artifact, 'url', 'success'));
  assert.equal(calls.filter(call => call.method === 'POST').length, 1);
  assert.ok(!calls.some(call => call.path.includes('statuses/')));
});
test('oversized reports fail explicitly instead of losing or truncating findings', async () => {
  const passes = roles.map((role, index) => ({ role, complete: true, findings: Array.from({ length: 4 }, (_, number) => ({
    id: `${role}-${number}`, severity: 'minor', path: 'a.go', line: 1, title: `Issue ${index}-${number}`,
    explanation: 'x'.repeat(6000), evidence: 'y'.repeat(6000) })) }));
  const calls = [];
  const api = async (path, method = 'GET', body) => { calls.push({ path, method, body }); return pr; };
  await publish(api, { ...artifact, passes }, 'url', 'success');
  assert.equal(calls.at(-1).body.state, 'error');
  assert.ok(calls.find(call => call.path.includes('/comments')).body.body.length < 60000);
});
test('valid clean report posts comment and exact-head success status', async () => {
  const calls = [];
  const api = async (path, method = 'GET', body) => { calls.push({ path, method, body }); return pr; };
  await publish(api, artifact, 'https://github.com/validflow/ccneed-cli/actions/runs/1', 'success');
  assert.equal(calls.filter(call => call.method === 'POST').length, 2);
  assert.deepEqual(calls.at(-1).body.state, 'success');
  assert.ok(calls.at(-1).path.endsWith(`statuses/${head}`));
});
test('failed job cannot publish green even if its artifact claims a clean report', async () => {
  const calls = [];
  const api = async (path, method = 'GET', body) => { calls.push({ path, method, body }); return pr; };
  await publish(api, artifact, 'url', 'failure');
  assert.equal(calls.at(-1).body.state, 'error');
});
test('malformed artifact records error rather than a success status', async () => {
  const calls = [];
  const api = async (path, method = 'GET', body) => { calls.push({ path, method, body }); return pr; };
  await publish(api, { ...artifact, passes: [] }, 'url', 'success');
  assert.equal(calls.at(-1).body.state, 'error');
});
test('fork preparation sets pending on head but does not fetch diff', async () => {
  const calls = [];
  const fork = { ...pr, head: { ...pr.head, repo: { full_name: 'outside/fork' } } };
  const api = async (path, method = 'GET', body) => {
    calls.push({ path, method, body });
    return path.includes('/permission') ? { permission: 'read' } : fork;
  };
  const result = await prepare(api, { number: 9, repository: 'validflow/ccneed-cli', event: 'pull_request_target', expectedHead: head, actor: 'member' }, 'url');
  assert.equal(result.allowed, false);
  assert.equal(calls.at(-1).body.state, 'pending');
  assert.equal(calls.some(call => call.path.includes('/files')), false);
});
