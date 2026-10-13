# 认证与环境

## 分两轮完成网页登录

第一轮发起授权，命令立即返回，不等待用户：

```bash
ccneed auth login --no-update-check
```

JSON 返回 `status: pending`、`origin`、`verification_url`、`user_code`、`expires_at`。原样展示链接、确认码和到期时间，请用户核对网站账号及确认码，在官网明确批准或拒绝。**结束本轮，把控制权交还用户**；不能展示链接后立即阻塞轮询，也不能替用户点击批准。

用户回复已授权后，在下一轮执行：

```bash
ccneed auth login --complete --timeout 60s --no-update-check
```

`--complete` 读取当前 origin 的系统凭证库证明；无需宿主保存设备秘密或把它传入参数。`--no-wait`、`--device-code`、`--token`、`--json` 不是 CCNeed 的参数。

| 返回结果 | 宿主下一步 |
| --- | --- |
| `status: authenticated` | 凭证已兑换并保存；结果还有 `origin`、`credential_id`、`user_id`、`expires_at`，继续用户原任务 |
| `status: pending, reason: timeout` | 只是本地等待到期，不是被拒绝；证明仍保存，未过期时可继续同一 `--complete` |
| `status: denied` | 用户已拒绝，短期证明清除；停止，不自动重新申请 |
| `status: expired` | 请求到期，短期证明清除；用户仍需登录时重新申请并展示新链接 |
| 非零退出码 | 按 `error.code` 处理，不声称已登录 |

**退出码 0 不保证授权成功，必须检查 `status`。** 本地等待默认 60 秒，`--timeout` 仅用于 `--complete`，范围 1 秒至 10 分钟，不延长授权请求的 10 分钟期限。CLI 自动遵守并保存服务器限频窗口，不用宿主高频循环替代。

`auth_request_missing` 或 `auth_request_expired` 表示本地没有可继续的证明，用户仍需登录时重新申请。网络失败或取消不能推断服务器拒绝；不自动更换环境或创建新请求。

## 本地状态与服务端验证

```bash
ccneed auth status --no-update-check
ccneed auth status --verify --no-update-check
```

默认 `status` 只检查本地记录，`state` 为 `missing`、`present` 或 `expired`，还包含所选 `origin`。已有记录可能包含 `credential_id`、`user_id`、`expires_at`，不含 token。

- `missing`：本 origin 没有凭证，不代表其他环境也没有。
- `present`：存在未过期记录，不代表服务器已验证有效。
- `expired`：本地凭证已到期，不能用于业务请求。
- `--verify`：只有当前 origin 的账号只读请求成功才增加 `authenticated: true`；失败为非零退出码，不伪造登录成功或零余额。

凭证库不可访问、记录损坏或操作取消不是普通“未登录”。系统解锁提示可能需要用户处理；取消本地等待不保证系统窗口立即关闭。

`auth login` 遇到 `credential_already_present` 不会覆盖已有有效凭证。先检查当前账号，正确时继续使用；用户明确要求换账号时才注销后重新授权。

## 退出与清理

```bash
ccneed auth logout --no-update-check
```

默认先撤销当前服务的客户端凭证，再清理本地凭证与短期证明；成功输出 `origin`、`removed: true`、`revoked: true`。不注销网站或其他设备。撤销失败不能声称已注销，不自动降级为本地清理。

仅用户明确要求只清本机，或无法访问服务器且理解风险时：

```bash
ccneed auth logout --local-only --no-update-check
```

此时 `revoked: false`：**服务端授权没有撤销**，而且丢失本地凭证后不能再用它发起撤销。不能静默用此命令替代正常退出。

不要导出网站 token、复制 Cookie 或手动往凭证库填入秘密来绕过授权。

## 开发环境

CLI 默认连接正式线上服务 `https://ccneed.ai`；普通用户无需传 `--api-url`。仅获授权的开发者联调 staging 时，才传自己确认可信的服务 origin 和 staging 测试账号；CLI 参数本身不授予环境访问权，服务端必须验证账号权限。通常传 HTTPS origin，不带 `/api` 路径、用户名、query 或 fragment；本机 loopback 可用 HTTP。若 staging 预览只有 HTTP 且使用 IP 地址，开发者可以在每条命令上额外传 `--allow-insecure-http`，但凭证会在网络中明文传输，绝不能对生产地址使用。该例外不改变默认 HTTPS 行为。

登录发起、完成、验证、注销和后续业务请求都保持同一 `--api-url`。官网链接从当前 origin 生成，应原样展示；不改写链接或到别的环境批准。凭证按 origin 隔离，不自动回退生产；网站登录态和 CLI 凭证是两回事。
