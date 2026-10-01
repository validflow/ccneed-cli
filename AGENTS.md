# AGENTS.md

项目通用指令以 [CLAUDE.md](CLAUDE.md) 为准。开始工作前完整阅读该文件及其要求的代码规范与贡献指南。

@CLAUDE.md

## Review guidelines

评审检查正确性、模块边界、安全、需求一致性四个方面。仅报告此次变更新增或放大的高置信度问题；高风险问题要二次核实，定位到具体变更。

重点检查凭证环境隔离、HTTP 目标可信度、异步任务与 HITL 幂等、文件下载安全、stdout/stderr 契约、真实行为测试和 Google 注释契约。

必需的自动审查由 GitHub Actions 的 CX Review 工作流调用仓库内 `.agents/skills/review-pipeline/SKILL.md`，不使用 `@codex review` 评论作为门禁。分别完成 Bug Detector、Architecture Compliance、Security Audit、Business Logic Validator 四个审查视角；Critical / Important 问题须二次核实。支持子 agent 时并行执行，否则明确记录四次独立审查的降级方式。即使没有发现问题，也输出总结；执行失败、缺失报告或无法验证不能报告通过。

CI 不在持有凭证的 runner 上构建或执行 PR 代码。外部贡献者 PR 需要维护者批准具体 head SHA 才能开始带模型凭证的审查；新提交使旧批准失效。评审结果必须对应当前 PR 的 head 与 base，不能复用过期结果。

遵循 CODE_STYLE.md，不把 Python runtime 的继承结构或本地 Agent 能力引入本仓。即使没有发现问题，也提供 review 总结。
