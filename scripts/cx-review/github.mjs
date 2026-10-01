import { readFile, writeFile, appendFile } from 'node:fs/promises';
import { pathToFileURL } from 'node:url';
import { admit, consolidate, render } from './gate.mjs';

// permission only downgrades a missing collaborator lookup; transport failures stay errors.
// Args: api is the GitHub boundary; login is a GitHub user or bot name.
// Returns: current collaborator permission or none for an explicit 404.
// Errors: invalid actor, authentication and transport errors are not swallowed.
async function permission(api, login) {
  if (!/^[a-zA-Z0-9-]+(?:\[bot\])?$/.test(login)) throw new Error('Invalid actor');
  try { return (await api(`collaborators/${encodeURIComponent(login)}/permission`)).permission; }
  catch (error) { if (error.status === 404) return 'none'; throw error; }
}

// prepare admits a fixed revision and always marks its required status pending first.
// Args: api performs GitHub operations; context binds event/actor/SHA; runURL identifies CI.
// Returns: admitted identity and the live metadata used for that decision.
// Errors: stale/unauthorized event or a failed pending-status write prevents admission.
export async function prepare(api, context, runURL) {
  if (!Number.isSafeInteger(context.number) || context.number < 1) throw new Error('Invalid PR number');
  const pr = await api(`pulls/${context.number}`);
  const authorPermission = await permission(api, pr.user.login);
  const actorPermission = context.event === 'workflow_dispatch' ? await permission(api, context.actor) : 'none';
  const decision = admit(pr, { ...context, authorPermission, actorPermission });
  await api(`statuses/${decision.head}`, 'POST', { state: 'pending', context: 'cx-review', target_url: runURL,
    description: decision.allowed ? 'CX Review queued for this exact head' : 'Maintainer must dispatch review for this exact head' });
  return { ...decision, metadata: pr };
}

// current checks both revision ends before publication; a changed PR is never greened.
// Args: pr is current live metadata; artifact contains expected number/head/base.
// Returns: nothing.
// Errors: throws on a closed PR or any changed identity/revision.
function current(pr, artifact) {
  if (pr.state !== 'open' || pr.number !== artifact.number || pr.head.sha !== artifact.head || pr.base.sha !== artifact.base) throw new Error('PR head or base changed; rerun review');
}

// publish validates independent outputs again and uses a separate workflow-only GitHub token.
// Args: api publishes; artifact is untrusted output with trusted identity; runURL links CI;
//   jobResult is the actual Actions job result, never model-provided status.
// Returns: published commit-status state.
// Errors: refuses stale publication; GitHub publication/transport errors remain failures.
export async function publish(api, artifact, runURL, jobResult) {
  const pr = await api(`pulls/${artifact.number}`);
  current(pr, artifact);
  let state = 'error';
  let report = '## CX Review\n\nReview failed or returned incomplete output. No passing review was recorded. See the workflow run.';
  if (jobResult === 'success') {
    try {
      const result = consolidate(artifact.passes, artifact.verification, artifact.files);
      const rendered = render(result, pr);
      if (Buffer.byteLength(rendered) > 60000) throw new Error('Report exceeds GitHub comment limit');
      state = result.state;
      report = rendered;
    } catch { /* An invalid report is a failing check, not an advisory warning. */ }
  }
  await api(`issues/${artifact.number}/comments`, 'POST', { body: report });
  // Recheck after posting the human report too, in case the head moved during that request.
  current(await api(`pulls/${artifact.number}`), artifact);
  await api(`statuses/${artifact.head}`, 'POST', { state, context: 'cx-review', target_url: runURL,
    description: state === 'success' ? 'Four passes complete; no confirmed blockers' : 'CX Review failed or has confirmed blockers' });
  return state;
}

// client limits GitHub authentication to the fixed API origin, with no redirects or raw body logging.
// Args: repository is owner/name; token is the job-scoped GitHub credential.
// Returns: a timeout-bounded API request function.
// Errors: missing configuration, failed/redirected requests or oversized/invalid JSON responses.
function client(repository, token) {
  if (!/^[a-zA-Z0-9_.-]+\/[a-zA-Z0-9_.-]+$/.test(repository) || !token) throw new Error('Missing GitHub configuration');
  return async (path, method = 'GET', body) => {
    const response = await fetch(`https://api.github.com/repos/${repository}/${path}`, { method, redirect: 'error',
      signal: AbortSignal.timeout(30000), headers: { Authorization: `Bearer ${token}`, Accept: 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28', 'Content-Type': 'application/json' }, body: body ? JSON.stringify(body) : undefined });
    if (!response.ok) { const error = new Error(`GitHub request failed (${response.status})`); error.status = response.status; throw error; }
    const text = await response.text();
    if (text.length > 4 * 1024 * 1024) throw new Error('GitHub response too large');
    return JSON.parse(text);
  };
}

// files collects every changed file rather than accepting an implicitly truncated first page.
// Args: api performs read-only GitHub requests; number is the admitted PR number.
// Returns: the full changed-file array.
// Errors: rejects malformed lists, excessive pagination and transport errors.
async function files(api, number) {
  const all = [];
  for (let page = 1; page <= 20; page++) {
    const batch = await api(`pulls/${number}/files?per_page=100&page=${page}`);
    if (!Array.isArray(batch)) throw new Error('Invalid changed file list');
    all.push(...batch);
    if (batch.length < 100) return all;
  }
  throw new Error('PR is too large for complete automated review');
}

// main separates authenticated orchestration from the tool-free model process.
// Returns: a promise completing the requested prepare or publish operation.
// Errors: unsupported operation, missing inputs, stale identity or I/O/API failures.
async function main() {
  const [command, output] = process.argv.slice(2);
  const api = client(process.env.GITHUB_REPOSITORY, process.env.GH_TOKEN);
  const runURL = `${process.env.GITHUB_SERVER_URL}/${process.env.GITHUB_REPOSITORY}/actions/runs/${process.env.GITHUB_RUN_ID}`;
  if (command === 'prepare') {
    const context = { number: Number(process.env.PR_NUMBER), repository: process.env.GITHUB_REPOSITORY,
      event: process.env.EVENT_NAME, expectedHead: process.env.EVENT_HEAD, approvedHead: process.env.APPROVED_HEAD,
      actor: process.env.GITHUB_ACTOR };
    const result = await prepare(api, context, runURL);
    await writeFile(output, JSON.stringify(result));
    await appendFile(process.env.GITHUB_OUTPUT, `allowed=${result.allowed}\nhead=${result.head}\nbase=${result.base}\nnumber=${result.number}\n`);
    if (result.allowed) {
      result.changed = await files(api, result.number);
      if (result.changed.length !== result.metadata.changed_files) throw new Error('Incomplete changed file list');
      current(await api(`pulls/${result.number}`), result);
      await writeFile(output, JSON.stringify(result));
    }
  } else if (command === 'publish') {
    const expected = JSON.parse(await readFile(process.env.PREPARED_FILE, 'utf8'));
    let artifact;
    try { artifact = JSON.parse(await readFile(output, 'utf8')); } catch { artifact = {}; }
    if (artifact.head !== expected.head || artifact.base !== expected.base || artifact.number !== expected.number) artifact = {};
    await publish(api, { ...artifact, head: expected.head, base: expected.base, number: expected.number,
      files: (expected.changed ?? []).map(file => file.filename) }, runURL, process.env.REVIEW_RESULT);
  } else { throw new Error('Unsupported CI operation'); }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main().catch(error => { console.error(error.message); process.exitCode = 1; });
}
