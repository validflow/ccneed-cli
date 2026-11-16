import { randomUUID } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { handleHook } from '../../skills/ccneed/scripts/acquisition.mjs';
import { hasCCNeedInvocation } from '../../skills/ccneed/scripts/command.mjs';

// The bundle patch mounts this Cordis module; these service names follow the
// native DSH tools, Skill registry and working-directory contracts.
export const name = 'ccneed';
export const inject = ['skills', 'tools', 'workingDirectory'];

/** hookInput translates one native Bash execution without inspecting unrelated tools.
 * Args:
 *   ctx: Native Cordis services scoped to this plugin.
 *   exec: DSH's immutable tool execution identity and parsed arguments.
 *   event: Shared hook event name.
 * Returns:
 *   A shared hook input, or null for an unrelated tool or command.
 */
function hookInput(ctx, exec, event) {
  if (exec.name !== 'bash' || !hasCCNeedInvocation(exec.arguments?.command)) return null;
  const session = exec.agent?.session;
  const directory = session ? ctx.workingDirectory.get(session) : process.cwd();
  const workdir = exec.arguments.workdir;
  return {
    hook_event_name: event,
    session_id: exec.agent?.id,
    cwd: typeof workdir === 'string' ? resolve(directory, workdir) : directory,
    tool_name: 'Bash',
    tool_input: { command: exec.arguments.command },
  };
}

/** checkedHook adapts native execution inputs to the shared business validation.
 * Args:
 *   ctx: Native Cordis services.
 *   exec: DSH tool execution.
 *   event: Shared event to process.
 *   options: Optional isolated hook state directory.
 *   response: Completed CLI output for post-execution handling.
 * Returns:
 *   Shared hook output, or an empty object for unrelated commands.
 */
function checkedHook(ctx, exec, event, options, response) {
  const input = hookInput(ctx, exec, event);
  return input ? handleHook('dsh', { ...input, tool_response: response }, options) : {};
}

/** apply registers the shared Skill and native per-call execution policies.
 * Args:
 *   ctx: Effect-owned native Cordis context; unloading removes every registration.
 *   options: Optional stateDir for isolated installations and runtime tests.
 * Errors:
 *   Missing native services or an incomplete bundled Skill fail plugin loading.
 */
export function apply(ctx, options = {}) {
  const path = fileURLToPath(new URL('../../skills/ccneed/SKILL.md', import.meta.url));
  const source = readFileSync(path, 'utf8');
  const match = /^---\r?\nname: ccneed\r?\ndescription: ([^\r\n]+)\r?\n---\r?\n([\s\S]*)$/.exec(source);
  if (!match) throw new Error('Incomplete bundled CCNeed Skill');
  ctx.skills.register({ name: 'ccneed', description: match[1], content: match[2], path, source: 'bundled', resourceBase: { kind: 'directory', path: fileURLToPath(new URL('../../skills/ccneed/', import.meta.url)).replace(/[\\/]$/, '') } });

  // A monotonic native guard prevents later policy listeners from overriding
  // a failed plan check. Native approvals remain owned by the DSH pipeline.
  ctx.tools.guard(exec => {
    const output = checkedHook(ctx, exec, 'PreToolUse', options).hookSpecificOutput;
    return output?.permissionDecision === 'deny' ? output.permissionDecisionReason : undefined;
  });
  ctx.on('tools/pre-execute', async (exec, next) => {
    if (exec.signal.aborted) return { kind: 'cancel' };
    const output = checkedHook(ctx, exec, 'PreToolUse', options).hookSpecificOutput;
    if (output?.permissionDecision === 'deny') return { kind: 'deny', reason: output.permissionDecisionReason };
    const prior = await next();
    if (prior.kind !== 'allow' || output?.permissionDecision !== 'ask') return prior;
    return { kind: 'ask', reason: output.permissionDecisionReason };
  });
  ctx.on('tools/post-execute', async (exec, result, next) => {
    const prior = await next();
    if (prior.kind !== 'accept' || result.isError || exec.signal.aborted) return prior;
    const response = Object.hasOwn(prior, 'value') ? prior.value : result.value;
    const stdout = response?.stdout;
    // Truncated or background output is never registered as a complete plan.
    const text = Object.hasOwn(prior, 'content') ? prior.content : stdout?.truncated ? '' : stdout?.text ?? result.content;
    const output = checkedHook(ctx, exec, 'PostToolUse', options, text).hookSpecificOutput;
    if (!output?.additionalContext) return prior;
    const context = { id: randomUUID(), role: 'user', source: { kind: 'ccneed-acquisition', form: 'instructions' }, content: [{ type: 'text', text: output.additionalContext }] };
    return { ...prior, additionalContexts: [...(prior.additionalContexts ?? []), context] };
  });
}
