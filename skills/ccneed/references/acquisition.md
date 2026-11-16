# 采集方案与确认

先检查安装版本的 `ccneed kb plan --help`、`ccneed kb start --help`。命令不存在或服务接口未部署时明确说明，不绕过成 HTTP 写请求。采集不会因为用户查看方案、余额或已有数据而开始。

## 直接获取数据

1. 宿主根据完整研究需求生成结构化方案；不调用网页的“优化意图/解析意图”。用户未指定数量时，所有平台合计默认 **300 条**；明确的总量或分平台数量优先。
2. 把方案写入普通 JSON 文件，用 `ccneed kb plan --plan-file ./plan.json --output ./prepared.json` 交平台校验、分配每词数量、计算积分并比较余额。这一步不创建知识库、不扣款、不采集。
3. Claude Code、Codex、WorkBuddy 的 PostToolUse 把最终方案送入上下文；Kimi 由 Agent 读取 CLI 完整输出和 `--output` 文件。每次请用户确认时，用自然语言按下方要求完整展示，不能因前文展示过而省略关键词，也不原样粘贴内部说明。保存完整返回 JSON，不能只存 token 或自行改展示字段；不用 shell 重定向隐藏 stdout。`invalid_plan` 的字段错误应修复后重新提交。
4. 等待用户在对话中确认或提出修改。用户可修改需求、平台、实体词、其他关键词或配额；两类关键词都可增删或替换，但每个平台至少保留一个有效实体词。修改后重新调用 `kb plan`，完整展示新结果再确认。`insufficient_credits`、无 token、过期或校验失败都不能执行。
5. 对话确认后，使用同一 prepared 文件执行 `ccneed kb start --plan-file ./prepared.json --request-id REQUEST_UUID`。PreToolUse 检查版本、有效期和余额，执行前只短提示，不重复整份方案。命令必须独立调用；权限弹窗或已有授权按下方宿主规则处理，Kimi 自动／会话授权继续有效。接收未知时保留同一 request ID，先查状态，不换 ID 重发。Codex、Kimi 的 prepared 和 HITL 回答文件必须用绝对路径；Kimi 的 `--output` 也使用新的绝对文件路径。
6. 按返回的真实 KB ID 查询状态；CLI 提交后由平台执行统一采集、打标流程，不另起打标任务。服务确认本次流程完成后，通常自动执行 `kb export --kb-id KB_ID --format csv` 并交付本地文件链接，无需再问是否导出。用户指定 JSON、两种格式或目录时按其要求；明确只需要云端分析时不额外导出。采集中、打标中继续有界查询同一 KB；失败或等待确认时说明实际状态，不以已接收、已有部分数据或云端回合结束代替任务完成。只需数据的用户不必创建云端分析会话或报告任务。

示例输入结构（使用用户的真实研究需求与词，不照抄样例）：

```json
{
  "research_requirement": "研究某品牌产品的用户体验与购买顾虑",
  "kb_name": "某品牌用户反馈",
  "content_scope": "both",
  "platforms": [
    {
      "platform": "xhs",
      "quota": 150,
      "keywords": [
        {"keyword": "某品牌产品", "tier": 1, "keyword_type": "entity"},
        {"keyword": "某品牌产品 使用体验", "tier": 1, "keyword_type": "preferred"},
        {"keyword": "某品牌产品 购买顾虑", "tier": 2, "keyword_type": "generated"}
      ]
    },
    {
      "platform": "weibo",
      "quota": 150,
      "keywords": [
        {"keyword": "某品牌产品", "tier": 1, "keyword_type": "entity"},
        {"keyword": "某品牌产品 评价", "tier": 2, "keyword_type": "generated"}
      ]
    }
  ]
}
```

补采集使用真实 `kb_id`，保留完整研究需求。平台负责验证所有权；不能因同名知识库猜 ID。

## Agent 生成的内部方案

T1/T2/T3 是宿主与平台的内部字段，不向用户解释术语或逐词数量。T1 包含实体词，以及与需求有关的用户偏好词。实体组必须非空，具体词可以修改。沿用平台分配：T1 占 80%，有实体和偏好词时内部为 30%/70%；剩余 20% 按 T2/T3 每词 3:2 分配，缺组时由现有组承接。平台完成最小配额、取整和总量校验，以返回方案为准。

实体词和其他关键词均可增删、替换，但每个平台的实体组不能删空；修改后由平台重新分配配额。新增实体词归实体组，新增的其他关键词归 T2，之后重新校验、报价、完整展示。不能为了修复 schema 或余额错误而静默丢掉研究对象、扩大预算或直接启动。

## 用户看到的方案

由 Agent 用自然语言说明研究需求，再按平台完整列出**实体词、其他关键词、整个平台多少条**；可用表格或列表，不要求固定话术或照抄 hook 的 Markdown。其他关键词包括 T1 偏好词与 T2/T3；不展示 tier、ratio、逐词 quota、token 或其他内部字段。再说明总条数、预计积分、当前余额及平台做过的缩量/修改说明。每次向用户发起方案确认，都应完整展示这些信息，不能只引用前文或省略关键词；正式执行前不再重复整份方案。

平台配额展示为“采集目标”，并说明平台按整页返回数据，实际采集条数可能超过目标；保留完整末页，不把目标描述成严格上限。

这是最终执行方案，两类关键词和配额仍可调整。对话确认的是方案内容，执行权限由宿主处理：需要审批时通过原生界面允许／拒绝，Kimi 已有的自动／会话授权按用户意愿生效。不能拿前一版展示作为新一版授权，也不以展示代替所需执行权限。

## 云端 Agent 对话中的采集

`tasks get/watch` 返回采集类 `waiting_user` 时，Agent 完整读取 `pending_action.suggested`，展示需求、平台、配额与积分并等待用户确认；Kimi 用绝对 `--output` 快照保留全文。按当前 `resume_schema` 保存答案，修改方案先重新校验。批准创建/补采集的 `tasks reply` 遵循相同宿主权限规则，Kimi 不重复拦截已有自动／会话授权。执行前只短提示；`user_feedback/user_rejected` 是反馈，不是批准。之后沿用同一回合取得真实 KB ID、等待采集完成并导出；缺少 KB ID 时说明情况，不按名称猜测。

其他 AskUser 澄清仍按[会话与任务](tasks.md)处理，不把所有 HITL 都改成采集授权。

## 宿主接入边界

安装/升级后重新加载宿主，并核对 hook 已加载、原生权限策略有效。CLI 本身是 Bash/执行命令工具运行的子进程，不能在 CLI 内伪造宿主权限弹窗。

- Claude Code：PreToolUse 可请求 `ask`；禁止用 bypass/dontAsk/自动批准模式执行采集。
- Codex：使用 `prefix_rule(..., decision="prompt")`。该规则保护受支持的沙箱外执行路径；付费命令必须通过人工审批的该路径执行。PreToolUse 的 `ask` 不受支持，不能返回它假装有效。禁用 hook/rules、忽略配置或自动审批时不得继续采集。
- WorkBuddy：桌面端使用 `~/.workbuddy` 的 Skill 与 hooks，付费采集使用同一宿主的**原生 Bash 工具**，保持默认权限与现有沙箱。沙箱内由安装器添加的 `sandbox.commandSafety.ask` 请求原生确认；当前 PowerShell 的安全删除包装不能可靠命中该规则，因此付费执行会拒绝并提示改用 Bash，不关闭沙箱。每次选择“仅允许本次”；选择“本会话不再询问”会主动扩大授权，需新建宿主会话恢复逐次确认。旧 `.codebuddy` 本地代码配置不代表桌面端已加载。

- Kimi Code：执行权限交给 Kimi 原生策略，尊重用户选择的询问、自动模式和会话授权；Hook 继续验证最终方案、积分与版本，不额外禁止自动模式。不要伪造 `ask`、权限字段或改用 HTTP 绕过业务校验。
- Kimi Code 2.5.3 的 SessionStart/PostToolUse 不注入 Hook 输出，且 Hook 只收到工具结果前 2000 字符。`kb plan` 与处理 HITL 的 `tasks get/watch` 使用新的绝对 `--output` 文件，由 Agent 读取完整方案，Hook 据同一文件校验。文件不能覆盖；本地 Hook 不读取远程 Bash 的文件，不根据截断内容推测方案。

所有宿主都不能以“配置文件存在”代替真实加载与权限验收，也不能在用户关闭 hook 或绕过全部权限后声称仍能强制确认。
