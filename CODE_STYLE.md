# 代码规范

沿用 usight_agent 的模块化、Google 注释结构、真实行为测试与干净代码原则；Go 语法遵循 [Google Go Style](https://google.github.io/styleguide/go/) 与 gofmt，不机械复制 Python 的格式或继承体系。

## 模块与依赖

- 每个包负责一件事；不建立 `utils` 大杂烩、万能 service 或巨型命令文件。
- 依赖向内：进程入口 → 命令层 → API/资源能力。低层不导入命令层，不导入 Club 或 usight_agent 的运行时。
- 外部 I/O 通过构造函数注入 HTTP client、输出 writer、凭证存储与 clock；只在确有替换需求的消费方定义小 interface，不给每个结构体创建一个 base interface。
- 不把登录、HTTP 请求、终端输出和任务状态机写在一个函数里。圈复杂度高于 10 由 lint 拦截；单文件过大、职责混杂由 review 判断。
- 默认没有后台 daemon、隐式全局状态或自动跨环境回退。

## 注释契约

所有生产函数、方法和类型都需要紧邻声明的注释。测试函数不强制重复签名说明；测试辅助函数应说明非显然行为。代码和注释不含 emoji、issue 编号或历史迁移标记。注释以中文为主，符号名和技术词保留英文。

Go 注释首句以符号名开始，再使用 Google 风格段落：有参数写 `Args:`；有返回值写 `Returns:`；返回 `error` 写 `Errors:`（Go 返回错误，不使用 Python 的 `Raises:`）。每个具名参数要说明含义、约束和单位；文档结构由 AST 检查，内容与真实行为是否一致由 review 验证。

发行与安装脚本使用 Node.js ES modules。导出的生产函数也说明输入、返回及失败条件；不要在脚本中拼接 shell 命令，外部进程使用参数数组。测试脚本可用简短的行为名，不必为每个断言补注释。

```go
// Submit 将任务提交到已认证的服务，不等待云端任务结束。
//
// Args:
//   ctx: 取消当前 HTTP 请求的上下文；取消不等于停止云端任务。
//   request: 已校验的提示词、资源标识与幂等请求标识。
//
// Returns:
//   创建的回合标识，或提交错误。
//
// Errors:
//   返回鉴权失败、参数错误或网络错误；不自动重试非幂等提交。
func (c *Client) Submit(ctx context.Context, request SubmitRequest) (Turn, error)
```

这是说明格式的例子，并非已实现的 API。不要描述自解释的逐行操作；重点写副作用、所有权、幂等性、超时和取消语义。包注释说明职责；已完成的公开 API 使用可运行的 `Example` 展示用法。

## 错误、并发与输出

- 错误显式向上传递，用 `%w` 保留原因；命令层决定用户展示与退出码，不在低层 `os.Exit`。
- `context.Context` 从调用方传入；所有网络操作有期限。超时不能谎报云端任务已取消。
- 非幂等写操作不做盲目重试；重复提交和 HITL 回答由服务端校验幂等及所有权。
- JSON 结果写 stdout，诊断写 stderr；不输出 token、完整认证响应或敏感请求头。
- 文件下载流式处理，并校验 artifact 所有权、文件名与目标路径；不静默覆盖已有文件。
- 凭证与网站会话不是同一份 cookie 文件；各环境凭证不得混用。

## 测试与检查

使用 Go 标准 testing、表驱动测试和 `httptest`。覆盖成功、边界、错误和副作用；跨模块链路尽量使用真实对象，只替换外部网络边界。并发测试不靠固定 sleep，不用删断言或 skip 来获得通过。

gofmt 决定格式；golangci-lint 检查 errcheck、govet、staticcheck、revive、ineffassign、unused、gocyclo 和 nolintlint。禁止 blanket nolint，必要豁免必须限定规则并说明理由。
