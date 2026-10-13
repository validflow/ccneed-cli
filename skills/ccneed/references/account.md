# 账号查询

读取当前凭证对应的账号，不接受其他 `user_id`，不修改积分或订阅，不发起付款。需要登录时先读 [认证与环境](auth.md)，完成官网授权后继续原请求。以下命令已随 CLI 发行并做了本地 HTTP 契约测试；若正式服务未提供对应接口，如实说明，不把本地通过当成线上可用。

| 用户需求 | 命令 | 主要 JSON 字段 |
| --- | --- | --- |
| 还有多少积分 | `ccneed account credits` | `left_credits` |
| 积分与有效订阅概览 | `ccneed account overview` | `credits.left_credits`、`subscriptions` |
| 最近 20 条积分记录 | `ccneed account history` | `items`、`scope`、`source_limit` |
| 调整输出条数 | `ccneed account history --limit 50` | 同上，范围 1–200 |

可追加 `--no-update-check`；staging 查询保持原来的 `--api-url`。

## 解释结果

- `left_credits` 是服务端权威余额，0 是有效值。读取失败不能回答“积分为 0”。
- `subscriptions` 包含有效订阅的 `plan_name`、`plan_level`、`expired_at`，字段可能为 null；缺失值不推断为永久有效。只有成功返回的空数组表示本次没有有效订阅。
- 记录包含 `trans_no`、`trans_type`，以及服务提供时的 `credits`、`created_at`、`expired_at`。积分记录可能为负数，不能据此自行重算余额。
- 历史仅覆盖最近 200 条窗口，`scope: recent`、`source_limit: 200`。`--limit` 截取本地输出，不是分页，不承诺完整历史。

需进一步网页操作时使用当前环境已确认的入口，不替用户充值、改套餐或切换账号。
