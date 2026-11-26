import { isAbsolute } from 'node:path';
import { handleHook } from '../skills/ccneed/scripts/acquisition.mjs';
import { hasCCNeedInvocation } from '../skills/ccneed/scripts/command.mjs';

/** invocation 将 OpenClaw shell 工具映射到共享 hook 输入，不处理其他工具。
 * Args:
 *   event: 原生工具调用。
 *   ctx: 原生会话及工作目录上下文。
 * Returns:
 *   识别出的输入；无关调用返回 null。
 */
function invocation(event, ctx) {
  if (!['exec', 'bash'].includes(event.toolName)) return null;
  const command = event.params?.command;
  if (typeof command !== 'string' || !hasCCNeedInvocation(command)) return null;
  return {
    session_id: ctx.sessionKey ?? ctx.sessionId,
    cwd: event.params.workdir ?? ctx.workspaceDir,
    tool_name: event.toolName,
    tool_input: { command },
  };
}

/** registerOpenClaw 接入原生结果事件并关联方案输出，不修改宿主配置。
 * Args:
 *   api: OpenClaw 原生插件注册接口。
 *   options: 测试可传入隔离的共享 hook 状态目录。
 */
export function registerOpenClaw(api, options) {
  const pending = new Map();
  api.on('before_tool_call', (event, ctx) => {
    const input = invocation(event, ctx);
    if (!input) return;
    const identified = typeof input.session_id === 'string' && input.session_id && event.toolCallId && typeof input.cwd === 'string' && isAbsolute(input.cwd);
    if (!identified) return;
    if (pending.size >= 1024) pending.delete(pending.keys().next().value);
    pending.set(JSON.stringify([input.session_id, event.toolCallId]), input);
  });
  api.on('tool_result_persist', (event, ctx) => {
    const key = JSON.stringify([ctx.sessionKey ?? ctx.sessionId, event.toolCallId]);
    const input = pending.get(key);
    if (!input || event.toolName !== input.tool_name) return;
    pending.delete(key);
    const result = handleHook('openclaw', { ...input, hook_event_name: 'PostToolUse', tool_response: event.isSynthetic || event.message?.isError ? null : event.message }, options);
    const context = result.hookSpecificOutput?.additionalContext;
    if (!context) return;
    return { message: { ...event.message, content: [...(event.message.content ?? []), { type: 'text', text: context }] } };
  });
  api.on('session_end', (_event, ctx) => {
    const session = ctx.sessionKey ?? ctx.sessionId;
    for (const [key, input] of pending) if (input.session_id === session) pending.delete(key);
  });
}

export default { id: 'ccneed', name: 'CCNeed', register: registerOpenClaw };
