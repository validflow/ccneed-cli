import { handleHook } from '../skills/ccneed/scripts/acquisition.mjs';

/** hookInput 将 Pi 的会话和 Bash 调用转换为共享校验协议。
 * Args:
 *   event: Pi 工具事件。
 *   ctx: 当前会话上下文。
 *   hook_event_name: 共享 hook 事件名。
 * Returns:
 *   不含额外会话或凭证数据的 hook 输入。
 */
function hookInput(event, ctx, hook_event_name) {
  return { hook_event_name, session_id: ctx.sessionManager.getSessionId(), cwd: ctx.cwd, tool_name: event.toolName, tool_input: event.input, tool_response: event.isError ? null : event.content };
}

/** registerPi 通过原生包扩展事件注入方案上下文，并使用 Pi 确认界面审批采集。
 * Args:
 *   pi: Pi 原生 ExtensionAPI。
 *   options: 测试可传入隔离的共享 hook 状态目录。
 * Errors:
 *   无交互界面或用户拒绝时阻止付费执行，保留业务校验的拒绝原因。
 */
export default function registerPi(pi, options) {
  pi.on('before_agent_start', (_event, ctx) => {
    const result = handleHook('pi', { hook_event_name: 'SessionStart', session_id: ctx.sessionManager.getSessionId(), cwd: ctx.cwd }, options);
    return { message: { customType: 'ccneed-context', content: result.hookSpecificOutput.additionalContext, display: false } };
  });
  pi.on('tool_call', async (event, ctx) => {
    if (event.toolName !== 'bash') return;
    const result = handleHook('pi', hookInput(event, ctx, 'PreToolUse'), options);
    const decision = result.hookSpecificOutput;
    if (decision?.permissionDecision === 'deny') return { block: true, reason: decision.permissionDecisionReason };
    if (decision?.permissionDecision !== 'ask') return;
    if (!ctx.hasUI) return { block: true, reason: 'CCNeed 正式采集需要 Pi 的原生确认界面；请在交互会话或支持扩展 UI 的 RPC 客户端执行。' };
    if (!await ctx.ui.confirm('CCNeed collection', decision.permissionDecisionReason)) return { block: true, reason: '用户取消了本次 CCNeed 采集。' };
  });
  pi.on('tool_result', (event, ctx) => {
    if (event.toolName !== 'bash') return;
    const result = handleHook('pi', hookInput(event, ctx, 'PostToolUse'), options);
    const context = result.hookSpecificOutput?.additionalContext;
    if (!context) return;
    return { content: [...event.content, { type: 'text', text: context }], ...(event.structuredContent === undefined ? {} : { structuredContent: event.structuredContent }) };
  });
}
