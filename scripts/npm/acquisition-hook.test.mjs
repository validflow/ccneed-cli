import assert from 'node:assert/strict';
import test from 'node:test';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { handleHook, renderPlan } from '../../hooks/acquisition.mjs';

function fixture() {
  const directory = mkdtempSync(join(tmpdir(), 'ccneed-hook-'));
  const plan = { status: 'ready', plan_token: 'server-token', plan_version: 'v1', expires_at: '2099-11-06T00:00:00Z', research_requirement: '比较品牌反馈', platforms: [{ platform: 'twitter', quota: 300, entity_keywords: ['Brand'], other_keywords: ['feedback'] }], search_plan: [{ platform: 'twitter', keyword: 'Brand', tier: 1, quota: 240, strategy_rationale: 'entity anchor' }, { platform: 'twitter', keyword: 'feedback', tier: 2, quota: 60, strategy_rationale: 'expanded' }], total_quota: 300, estimated_credits: 300, balance: 1000 };
  const filename = join(directory, 'prepared.json');
  writeFileSync(filename, JSON.stringify(plan));
  const input = { session_id: 'same-session', cwd: directory, tool_name: 'Bash', permission_mode: 'default', approvals_reviewer: 'user', approval_policy: 'on-request' };
  return { directory, plan, filename, input, options: { stateDir: join(directory, 'state') } };
}

const quoted = value => '"' + value.replaceAll('\\', '/') + '"';

for (const host of ['claude', 'codex', 'workbuddy']) {
  test(host + ' leaves literal native plugin queries and their output untouched', () => {
    const data = fixture();
    for (const command of [
      'codex plugin list --marketplace ccneed --json',
      'codex.cmd plugin list --json -m ccneed --available',
      'codex.exe plugin list --marketplace=ccneed --json',
      'Codex.CMD plugin list --marketplace "ccneed"',
    ]) {
      for (const hook_event_name of ['PreToolUse', 'PostToolUse']) {
        const event = { ...data.input, hook_event_name,
          tool_name: host === 'codex' ? 'exec_command' : 'Bash',
          tool_input: host === 'codex' ? { cmd: command } : { command },
          tool_response: { stdout: '{"installed":[]}' } };
        assert.deepEqual(handleHook(host, event, data.options), {}, command);
      }
    }
  });
}

test('wrapped or compound collection calls still fail after a plan has been displayed', () => {
  const data = fixture();
  for (const host of ['claude', 'codex', 'workbuddy']) {
    handleHook(host, { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file raw.json' }, tool_response: data.plan }, data.options);
    const start = 'ccneed kb start --plan-file ' + quoted(data.filename);
    for (const command of [
      'env ' + start,
      'bash -lc ' + quoted(start),
      '/usr/local/bin/' + start,
      'codex plugin list --marketplace ccneed --json; ' + start,
      'codex plugin list --marketplace ccneed --config profile=other',
      start + ' | cat',
    ]) {
      const result = handleHook(host, { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command } }, data.options);
      assert.equal(result.hookSpecificOutput?.permissionDecision, 'deny', command);
    }
  }
});

test('plan presentation groups whole platform quota and hides internal allocation', () => {
  const { plan } = fixture();
  const text = renderPlan(plan);
  for (const value of ['比较品牌反馈', 'Brand', 'feedback', '300', '1000']) assert.ok(text.includes(value));
  for (const value of ['T1', 'T2', 'tier', '240', '60', 'server-token']) assert.ok(!text.includes(value));
});

for (const host of ['claude', 'codex', 'workbuddy']) {
  test(host + ' explains page-sized collection targets for direct and cloud plans', () => {
    const data = fixture();
    const suggested = { mode: 'add', kb_id: 7, kb_description: data.plan.research_requirement, quotas: { twitter: 300 }, search_plan: data.plan.search_plan, estimated_credits: 300, balance: 1000 };
    const events = [
      { command: 'ccneed kb plan --plan-file raw.json', response: data.plan },
      { command: 'ccneed tasks get --thread-id thread --turn-id turn', response: { thread_id: 'thread', turn_id: 'turn', pending_action: { action_id: 'action', suggested } } },
    ];
    for (const { command, response } of events) {
      const result = handleHook(host, { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command }, tool_response: response }, data.options);
      const context = result.hookSpecificOutput.additionalContext;
      assert.ok(context.includes('采集目标（条）'));
      assert.ok(context.includes('整页'));
      assert.ok(context.includes('可能超过目标'));
      assert.ok(!context.includes('采集上限'));
      for (const value of ['比较品牌反馈', 'Brand', 'feedback', '300', '1000']) assert.ok(context.includes(value));
      assert.equal(result.systemMessage, undefined);
    }
  });
}

for (const host of ['claude', 'codex', 'workbuddy']) {
  test(host + ' rejects a plan without a nonblank entity and clears the old approval candidate', () => {
    for (const entities of [[], [''], [' \t ']]) {
      const data = fixture();
      const post = { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file raw.json' } };
      handleHook(host, { ...post, tool_response: data.plan }, data.options);
      data.plan.platforms[0].entity_keywords = entities;
      writeFileSync(data.filename, JSON.stringify(data.plan));
      handleHook(host, { ...post, tool_response: data.plan }, data.options);
      const start = handleHook(host, { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed kb start --plan-file ' + quoted(data.filename) } }, data.options);
      assert.equal(start.hookSpecificOutput.permissionDecision, 'deny');
    }
  });

  test(host + ' accepts replaced entities and editable other keywords only after re-preparation', () => {
    for (const other of [['support experience'], []]) {
      const data = fixture();
      const post = { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file raw.json' } };
      const start = { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed kb start --plan-file ' + quoted(data.filename) } };
      handleHook(host, { ...post, tool_response: data.plan }, data.options);
      data.plan.plan_version = 'v2';
      data.plan.platforms[0].entity_keywords = ['NextProduct', 'NextAlias'];
      data.plan.platforms[0].other_keywords = other;
      data.plan.search_plan = [
        { platform: 'twitter', keyword: 'NextProduct', tier: 1, quota: 150, strategy_rationale: 'entity anchor' },
        { platform: 'twitter', keyword: 'NextAlias', tier: 1, quota: other.length ? 90 : 150, strategy_rationale: 'entity anchor' },
        ...(other.length ? [{ platform: 'twitter', keyword: other[0], tier: 2, quota: 60, strategy_rationale: 'expanded' }] : []),
      ];
      writeFileSync(data.filename, JSON.stringify(data.plan));
      assert.equal(handleHook(host, start, data.options).hookSpecificOutput.permissionDecision, 'deny');
      for (let review = 0; review < 2; review++) {
        const context = handleHook(host, { ...post, tool_response: data.plan }, data.options).hookSpecificOutput.additionalContext;
        for (const value of ['NextProduct', 'NextAlias', ...other, '300', '1000']) assert.ok(context.includes(value));
      }
      const permission = handleHook(host, start, data.options).hookSpecificOutput;
      if (host === 'codex') assert.equal(permission.permissionDecision, undefined);
      else assert.equal(permission.permissionDecision, 'ask');
    }
  });
}

for (const host of ['claude', 'codex', 'workbuddy']) {
  test(host + ' requires same-session final presentation before native confirmation', () => {
    const data = fixture();
    const start = { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed kb start --plan-file ' + quoted(data.filename) + ' --request-id 00000000-0000-0000-0000-000000000222' } };
    assert.equal(handleHook(host, start, data.options).hookSpecificOutput.permissionDecision, 'deny');
    const shown = handleHook(host, { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file raw.json' }, tool_response: { stdout: JSON.stringify(data.plan) } }, data.options);
    assert.equal(shown.systemMessage, undefined);
    assert.equal(shown.hookSpecificOutput.hookEventName, 'PostToolUse');
    const context = shown.hookSpecificOutput.additionalContext;
    for (const value of ['比较品牌反馈', 'Brand', 'feedback', '300', '1000']) assert.ok(context.includes(value));
    for (const value of ['server-token', 'tier', '240']) assert.ok(!context.includes(value));
    const allowed = handleHook(host, start, data.options);
    assert.notEqual(allowed.hookSpecificOutput?.permissionDecision, 'deny');
    if (host !== 'codex') assert.equal(allowed.hookSpecificOutput.permissionDecision, 'ask');
    assert.ok(allowed.systemMessage.includes('300'));
    for (const value of ['比较品牌反馈', 'Brand', 'feedback', '|', '###']) assert.ok(!allowed.systemMessage.includes(value));
    data.plan.platforms[0].quota = 301;
    writeFileSync(data.filename, JSON.stringify(data.plan));
    assert.equal(handleHook(host, start, data.options).hookSpecificOutput.permissionDecision, 'deny');
  });
}

for (const host of ['claude', 'codex', 'workbuddy']) {
  test(host + ' sends invalid-plan feedback to the Agent and invalidates the earlier plan', () => {
    const data = fixture();
    const event = { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file raw.json' } };
    handleHook(host, { ...event, tool_response: data.plan }, data.options);
    const result = handleHook(host, { ...event, tool_response: { status: 'invalid_plan', errors: [{ field: 'platforms.0.quota', message: 'Quota must be positive' }] } }, data.options);
    assert.equal(result.systemMessage, undefined);
    assert.equal(result.hookSpecificOutput.hookEventName, 'PostToolUse');
    assert.ok(result.hookSpecificOutput.additionalContext.includes('platforms\\.0\\.quota'));
    assert.ok(result.hookSpecificOutput.additionalContext.includes('Quota must be positive'));
    const denied = handleHook(host, { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed kb start --plan-file ' + quoted(data.filename) } }, data.options);
    assert.equal(denied.hookSpecificOutput.permissionDecision, 'deny');
  });

  test(host + ' sends cloud collection plans to Agent context before requesting native approval', () => {
    const data = fixture();
    const suggested = { mode: 'add', kb_id: 7, kb_description: data.plan.research_requirement, quotas: { twitter: 300 }, search_plan: data.plan.search_plan, estimated_credits: 300, balance: 1000 };
    const action = { thread_id: 'thread', turn_id: 'turn', pending_action: { action_id: 'action', suggested } };
    const result = handleHook(host, { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed tasks get --thread-id thread --turn-id turn' }, tool_response: action }, data.options);
    assert.equal(result.systemMessage, undefined);
    assert.equal(result.hookSpecificOutput.hookEventName, 'PostToolUse');
    assert.ok(result.hookSpecificOutput.additionalContext.includes('比较品牌反馈'));
    assert.ok(result.hookSpecificOutput.additionalContext.includes('Brand'));
    const filename = join(data.directory, 'answer.json');
    writeFileSync(filename, '{"decision":"approved"}');
    const approval = handleHook(host, { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed tasks reply --thread-id thread --turn-id turn --action-id action --answer-file ' + quoted(filename) } }, data.options);
    assert.notEqual(approval.hookSpecificOutput?.permissionDecision, 'deny');
    if (host !== 'codex') assert.equal(approval.hookSpecificOutput.permissionDecision, 'ask');
    for (const value of ['比较品牌反馈', 'Brand', '|', '###']) assert.ok(!approval.systemMessage.includes(value));
  });

  test(host + ' still rejects expired and unaffordable plans without a native start', () => {
    for (const update of [{ expires_at: '2000-01-01T00:00:00Z' }, { balance: 299 }, { estimated_credits: null }]) {
      const data = fixture();
      Object.assign(data.plan, update);
      writeFileSync(data.filename, JSON.stringify(data.plan));
      handleHook(host, { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file raw.json' }, tool_response: data.plan }, data.options);
      const result = handleHook(host, { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed kb start --plan-file ' + quoted(data.filename) } }, data.options);
      assert.equal(result.hookSpecificOutput.permissionDecision, 'deny');
    }
  });
}

test('known bypass mode cannot perform a paid crawl', () => {
  const data = fixture();
  handleHook('claude', { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file raw.json' }, tool_response: data.plan }, data.options);
  const result = handleHook('claude', { ...data.input, permission_mode: 'bypassPermissions', hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed kb start --plan-file ' + quoted(data.filename) } }, data.options);
  assert.equal(result.hookSpecificOutput.permissionDecision, 'deny');
});

test('noncanonical executable case cannot bypass native prefix rules', () => {
  const data = fixture();
  handleHook('codex', { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file raw.json' }, tool_response: data.plan }, data.options);
  const result = handleHook('codex', { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command: 'CCNEED.CMD kb start --plan-file ' + quoted(data.filename) } }, data.options);
  assert.equal(result.hookSpecificOutput?.permissionDecision, 'deny');
});

test('help text consumed as a file value does not disable confirmation guards', () => {
  const data = fixture();
  for (const command of ['ccneed kb start --plan-file --help --request-id id', 'ccneed --no-update-check kb start --plan-file --help --request-id id', 'ccneed tasks reply --answer-file -h --thread-id thread --turn-id turn --action-id action']) {
    const result = handleHook('codex', { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command } }, data.options);
    assert.equal(result.hookSpecificOutput?.permissionDecision, 'deny');
  }
});

test('WorkBuddy paid PowerShell commands cannot use an unverified sandbox approval path', () => {
  const data = fixture();
  handleHook('workbuddy', { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed kb plan --plan-file raw.json' }, tool_response: data.plan }, data.options);
  const result = handleHook('workbuddy', { ...data.input, tool_name: 'PowerShell', hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed kb start --plan-file ' + quoted(data.filename) } }, data.options);
  assert.equal(result.hookSpecificOutput?.permissionDecision, 'deny');
});

test('cloud HITL accepts the displayed final plan and preserves feedback without starting', () => {
  const data = fixture();
  const suggested = { mode: 'create', kb_name: 'Brand', kb_description: data.plan.research_requirement, keywords: ['original preferred seed'], quotas: { twitter: 300 }, content_scope: 'all', search_plan: data.plan.search_plan, plan_token: 'keyword-token', estimated_credits: 300, balance: 1000 };
  const action = { thread_id: 'thread', turn_id: 'turn', pending_action: { action_id: 'action', action_type: 'tool_approval', suggested } };
  handleHook('workbuddy', { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed tasks watch --thread-id thread --turn-id turn' }, tool_response: { stdout: JSON.stringify(action) } }, data.options);
  const filename = join(data.directory, 'answer.json');
  const input = { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed tasks reply --thread-id thread --turn-id turn --action-id action --answer-file ' + quoted(filename) } };
  writeFileSync(filename, JSON.stringify({ decision: 'approved', modified_input: { keywords: ['Brand'], quotas: suggested.quotas, search_plan: suggested.search_plan, keyword_plan_token: 'keyword-token' } }));
  assert.equal(handleHook('workbuddy', input, data.options).hookSpecificOutput.permissionDecision, 'ask');
  writeFileSync(filename, JSON.stringify({ decision: 'approved', modified_input: { keywords: ['Brand'], quotas: suggested.quotas, search_plan: suggested.search_plan, keyword_plan_token: 'keyword-token', user_reply: 'Focus on product experience' } }));
  assert.equal(handleHook('workbuddy', input, data.options).hookSpecificOutput.permissionDecision, 'ask');
  writeFileSync(filename, JSON.stringify({ decision: 'approved', response_kind: 'user_feedback', reply: '减少到100条' }));
  assert.deepEqual(handleHook('workbuddy', input, data.options), {});
  writeFileSync(filename, JSON.stringify({ decision: 'rejected' }));
  assert.deepEqual(handleHook('workbuddy', input, data.options), {});
});

test('cloud approval refuses an unavailable or insufficient wallet quotation', () => {
  const data = fixture();
  const suggested = { mode: 'create', kb_description: data.plan.research_requirement, quotas: { twitter: 300 }, search_plan: data.plan.search_plan, estimated_credits: 300, balance: null };
  handleHook('workbuddy', { ...data.input, hook_event_name: 'PostToolUse', tool_input: { command: 'ccneed tasks get --thread-id thread --turn-id turn' }, tool_response: { thread_id: 'thread', turn_id: 'turn', pending_action: { action_id: 'action', suggested } } }, data.options);
  const filename = join(data.directory, 'answer.json');
  writeFileSync(filename, '{"decision":"approved"}');
  const result = handleHook('workbuddy', { ...data.input, hook_event_name: 'PreToolUse', tool_input: { command: 'ccneed tasks reply --thread-id thread --turn-id turn --action-id action --answer-file ' + quoted(filename) } }, data.options);
  assert.equal(result.hookSpecificOutput.permissionDecision, 'deny');
});
