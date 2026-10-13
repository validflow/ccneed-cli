# 会话、回合与用户确认

以下命令已随 CLI 发行并有本地契约测试；具体云端能力仍取决于正式服务接口是否上线。示例中的 `THREAD_UUID`、`TURN_UUID`、`REQUEST_UUID`、`ACTION_ID` 必须替换成当前环境真实返回的标识，不能照抄或从会话标题推导。

## 开启或选择会话

```bash
ccneed threads list --view active --limit 50
ccneed run "分析已选资料并生成报告" --kb-id KB_ID
```

`--view` 为 `active` 或 `archived`，`--limit` 范围 1–200，仅显示近期窗口。标题用来帮助用户选择，后续调用以真实 `thread_id` 为准；同名会话有歧义时问用户。

开启新会话时把第一句话和已有 `--kb-id`、`--file-id` 一起交给 `run`；`--title` 可选，未指定时服务端取首句作为标题。也可只绑定已上传文件，详见 [资源与产出](resources.md)。接收状态未知时先查列表/网站，不能盲目再开启。

## 在已有会话提交一轮

```bash
ccneed run "分析已绑定资料并生成报告" --thread-id THREAD_UUID
```

输入是用户此次明确委托的内容。首次 `run` 和后续 `run --thread-id` 都返回 `status: accepted`、`cloud_status`、`thread_id`、`turn_id`、`client_request_id`、`conversation_turn_idx`、`existing`，即云端已接收，不是分析完成。保存公开标识以继续观察同一回合；后续消息可在同一会话提交新回合，HITL 回答则用 `tasks reply` 而不是新建回合。

CLI 在单次提交前保存按 origin/thread 隔离的不可变 request ID 与输入摘要，不保存输入正文或凭证。

若 `error.code: acceptance_unknown`，接收情况不明，读取 `recovery.thread_id`、`recovery.client_request_id`。确需恢复提交时只能显式沿用原 ID 和**完全相同的输入**：

```bash
ccneed run "原始完整输入" --thread-id THREAD_UUID --request-id REQUEST_UUID
```

不能更换 ID 自动重发；同一 ID 不同输入会 `request_id_conflict`。拿到回合 ID 后查询该回合；无法确认原输入或身份时向用户说明，不猜测恢复。

## 历史与终端退出后的恢复

```bash
ccneed tasks list --thread-id THREAD_UUID
ccneed tasks list --thread-id THREAD_UUID --request-id REQUEST_UUID
ccneed requests list --thread-id THREAD_UUID
```

`tasks list` 是只读历史页，返回原始回合标识、顺序、提交标识和 `cloud_status`；默认 20，`--limit` 范围 1–100。下一页原样传 `next_before_turn_index` 为 `--before-turn-index`。用目标 `turn_id` 再 `get` 获取该轮回答，不能把最近一页说成完整历史。

确认响应丢失时先用 `--request-id` 找原回合，不重提任务。终端未能输出标识时，`requests list` 找本地原 ID，只返回当前 origin/thread 的非秘密引用；`cloud_acceptance: unknown` 不是服务已接收。无法读出恢复记录时说明错误，不造新 ID 规避。查询无匹配也不代表原请求永远不会被云端接收，仍遵守原 ID/原输入的恢复约束。

## 查询与有界观察

```bash
ccneed tasks get --thread-id THREAD_UUID --turn-id TURN_UUID
ccneed tasks watch --thread-id THREAD_UUID --turn-id TURN_UUID --timeout 60s
```

两者返回目标回合的 `status`、`cloud_status`、公开 `answer`、`answer_truncated`、`pending_action`、`artifacts`、`artifacts_has_more`，并保留会话/回合标识。`watch` 额外返回 `stop_reason`。

- `waiting_user`：立即把待确认项交给用户，不继续等待。
- `completed`：交付这一轮的公开回答与关联产出；`answer_truncated` 为真时不能声称回答完整。
- `failed`、`cancelled`、`interrupted`：说明实际状态，不声称已完成，也不自动重新提交。
- `stop_reason: timeout`：只停止本地观察，不取消云端任务；稍后仍查同一回合。首次查询未得到快照时状态可能是 `unknown`，不能猜成正在运行或完成。

`watch` 默认 60 秒，期限范围 1 秒至 10 分钟；不是后台 daemon，也不是用户确认轮询。退出码 0 不表示云端任务成功，必须检查返回状态。此接口不展示隐藏推理、原始工具循环或整段对话全文。

## 通用 HITL 回答

`pending_action` 包含 `action_id`、`action_type`、`resume_schema`，以及可选 `reason`、`suggested`。向用户说明问题/选项、影响和所需参数，按实际 schema 收集选择；建议值不等于用户同意。不要假设每种 action 都只有一个 `approved` 字段。

例如 AskUser 返回 `reason: 用户想研究的“苹果”具体指哪一个？` 和 `suggested.options: [苹果公司, 水果苹果]` 时，先把这两个选项发给用户，结束当前回复并等待。用户说“苹果公司，我想研究 iPhone 18”后，若 `resume_schema` 要求 `reply` 字符串，就把这段完整原话放入 `{"reply":"苹果公司，我想研究 iPhone 18"}`；不要只取选项、替用户挑选，或把 HITL 回答另发成 `ccneed run`。其他 action 仍以它们各自的 schema 为准。

用户确认后，把满足当前 `resume_schema` 的 JSON 对象写入普通文件（不超过 64 KiB），再提交：

```bash
ccneed tasks reply --thread-id THREAD_UUID --turn-id TURN_UUID --action-id ACTION_ID --answer-file ./answer.json
```

`status: queued` 只表示恢复请求排队，之后继续 `get/watch` 同一回合，不开启新回合。接收未知先查询，不自动重发；action 已过期、被其他客户端回答或状态改变时重新读取快照，不套用旧选择。schema 错误只修正答案，不绕过用户确认。

若原任务明确要求确认后只复述选择，回合完成后不能声称已执行用户后来提到的研究；需要新的实际研究任务时，向用户说明当前结果，并在用户明确委托后再提交下一回合。

网页登录授权与任务 HITL 是两个流程：前者使用 `auth login`，后者使用本回合的 `tasks reply`。
