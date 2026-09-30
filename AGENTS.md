# AGENTS.md

项目通用指令以 [CLAUDE.md](CLAUDE.md) 为准。开始工作前完整阅读该文件及其要求的代码规范与贡献指南。

@CLAUDE.md

## Review guidelines

评审检查正确性、模块边界、安全、需求一致性四个方面。仅报告此次变更新增或放大的高置信度问题；高风险问题要二次核实，定位到具体变更。

重点检查凭证环境隔离、HTTP 目标可信度、异步任务与 HITL 幂等、文件下载安全、stdout/stderr 契约、真实行为测试和 Google 注释契约。

遵循 CODE_STYLE.md，不把 Python runtime 的继承结构或本地 Agent 能力引入本仓。即使没有发现问题，也提供 review 总结。
