# CCNeed CLI 开发指令

本文件是项目指令的单一来源。开始工作前完整阅读 [CODE_STYLE.md](CODE_STYLE.md) 和 [CONTRIBUTING.md](CONTRIBUTING.md)。

- Go 完整命令行接入工具，覆盖登录、资源、长程任务、确认及结果生命周期；云端执行任务，不复制 CCNeed runtime。
- 产品命令、API、登录凭证、资源处理和输出分层；只为当前需求增加模块。
- 所有生产函数、方法和类型有符号名开头的 Google 风格注释；遵守 Args / Returns / Errors 契约。代码与注释不放 issue 编号或 emoji。
- 新增行为先写失败测试并实跑，再实现到绿；禁止空壳、弱化断言或无理由 skip。
- 提交前执行 `make check` 或 `bash scripts/check.sh`，不得跳过失败步骤。
- 普通用户只使用 sg-prod；开发联调 staging。数据库、凭证、存储和任务副作用严格隔离，不自动回退生产。
- 初始基础提交之后，main 只通过功能分支 PR、review、质量门禁、squash merge 更新；不要创建 develop。
- README 只描述已可用能力；规划能力明确标注，不给未发布包提供貌似可用的安装命令。
- 不写 token、账号、内部编排 UUID、私有 API 样例响应或未确认的生产域名。许可证与发布需单独授权。
- 文档保持简短、按读者划分；README 面向用户，CONTRIBUTING 面向贡献者，CODE_STYLE 管代码契约。不引入服务仓的文档索引系统。
