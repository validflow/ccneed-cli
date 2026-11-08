# 采集方案与确认

先检查安装版本的 `ccneed kb plan --help`、`ccneed kb start --help`。命令不存在或服务接口未部署时明确说明，不绕过成 HTTP 写请求。采集不会因为用户查看方案、余额或已有数据而开始。

## 直接获取数据

1. 宿主根据完整研究需求生成结构化方案；不调用网页的“优化意图/解析意图”。用户未指定数量时，所有平台合计默认 **300 条**；明确的总量或分平台数量优先。
2. 把方案写入普通 JSON 文件，用 `ccneed kb plan --plan-file ./plan.json --output ./prepared.json` 交平台校验、分配每词数量、计算积分并比较余额。这一步不创建知识库、不扣款、不采集。
3. plugin 的 PostToolUse hook 把平台返回的最终方案作为上下文交给 Agent。Agent 用自然语言按下方要求向用户展示一次，不原样粘贴内部说明；`--output` 同时完整保存返回 JSON，不能只保存 token 后自行改展示字段。stdout 保留给 hook，不用 shell 重定向隐藏方案。`invalid_plan` 的字段错误应修复后重新提交，不把校验失败当作已准备好。
4. 等待用户在对话中确认或提出修改。用户可修改需求、平台、关键词或配额；修改后重新调用 `kb plan`，展示新结果再确认。`insufficient_credits`、无 token、过期或校验失败都不能执行。
5. 对话确认后，使用同一 prepared 文件执行 `ccneed kb start --plan-file ./prepared.json --request-id REQUEST_UUID`。PreToolUse 检查版本、有效期和余额，只短提示“即将按上述已确认方案开始采集”，不重复整份方案。命令必须独立调用，由宿主原生权限 UI 请求本次执行授权；拒绝即不执行。接收未知时保留同一 request ID，先查状态，不换 ID 重发。Codex 的 prepared 方案和 HITL 回答文件必须使用绝对路径；建议准备方案时 `--output` 也指定绝对路径。
6. 按返回的真实 KB ID 查询采集状态；服务确认采集完成后，通常自动执行 `kb export --kb-id KB_ID --format csv` 并交付本地文件链接，无需再问是否导出。用户指定 JSON、两种格式或目录时按其要求；明确只需要云端分析时不额外导出。采集中继续有界查询同一 KB；失败或等待确认时说明实际状态，不以已接收、已有部分数据或云端回合结束代替采集完成。只需数据的用户不必创建云端分析会话、打标需求或报告任务。

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

T1/T2/T3 是宿主与平台的内部字段，不向用户解释术语或逐词数量。T1 包含必须保留的实体词，以及与需求有关的用户偏好词。沿用平台分配：T1 占 80%，有实体和偏好词时内部为 30%/70%；剩余 20% 按 T2/T3 每词 3:2 分配，缺组时由现有组承接。平台完成最小配额、取整和总量校验，以返回方案为准。

删除偏好词或其他关键词后，由平台重新平摊配额；实体锚点不能删除。用户新加词归 T2，之后重新校验、报价、展示。不能为了修复 schema 或余额错误而静默丢掉研究对象、扩大预算或直接启动。

## 用户看到的方案

由 Agent 用自然语言展示完整研究需求，再按平台清楚列出**实体词、其他关键词、整个平台多少条**；需要对比时可以用简短表格，不要求照抄 hook 的 Markdown。其他关键词包括 T1 偏好词与 T2/T3；不展示 tier、ratio、逐词 quota、token 或其他内部字段。再说明总条数、预计积分、当前余额及平台做过的缩量/修改说明。同一份方案只展示一次，执行前不再重复整段内容。

这是最终执行方案。用户可继续调整；原生权限确认紧接实际执行，不能拿前一版展示为新一版授权。展示成功也不等于用户已经授权。

## 云端 Agent 对话中的采集

`tasks get/watch` 返回采集类 `waiting_user` 时，plugin 同样把 `pending_action.suggested` 送入上下文，由宿主 Agent 展示研究需求、平台、配额及积分并等待用户确认。按当前 `resume_schema` 保存答案，修改方案先重新校验。真正批准创建/补采集的 `tasks reply` 也须触发原生权限，执行前只显示短提示；`response_kind: user_feedback/user_rejected` 是反馈，不应当成批准执行。批准后沿用同一回合观察结果，从公开回复取得真实 KB ID，再查询采集状态；采集完成后按上述默认格式导出并交付，拿不到 KB ID 时说明缺失，不按名称猜测。

其他 AskUser 澄清仍按[会话与任务](tasks.md)处理，不把所有 HITL 都改成采集授权。

## 宿主接入边界

安装/升级后重新加载宿主，并核对 hook 已加载、原生权限策略有效。CLI 本身是 Bash/执行命令工具运行的子进程，不能在 CLI 内伪造宿主权限弹窗。

- Claude Code：PreToolUse 可请求 `ask`；禁止用 bypass/dontAsk/自动批准模式执行采集。
- Codex：使用 `prefix_rule(..., decision="prompt")`。该规则保护受支持的沙箱外执行路径；付费命令必须通过人工审批的该路径执行。PreToolUse 的 `ask` 不受支持，不能返回它假装有效。禁用 hook/rules、忽略配置或自动审批时不得继续采集。
- WorkBuddy：桌面端使用 `~/.workbuddy` 的 Skill 与 hooks，付费采集使用同一宿主的**原生 Bash 工具**，保持默认权限与现有沙箱。沙箱内由安装器添加的 `sandbox.commandSafety.ask` 请求原生确认；当前 PowerShell 的安全删除包装不能可靠命中该规则，因此付费执行会拒绝并提示改用 Bash，不关闭沙箱。每次选择“仅允许本次”；选择“本会话不再询问”会主动扩大授权，需新建宿主会话恢复逐次确认。旧 `.codebuddy` 本地代码配置不代表桌面端已加载。

三个宿主都不能以“配置文件存在”代替真实加载与权限验收，也不能在用户关闭 hook 或绕过全部权限后声称仍能强制确认。
