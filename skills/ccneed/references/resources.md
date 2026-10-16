# 资源与产出

以下命令已随 CLI 发行并有本地契约测试；具体云端能力仍取决于正式服务接口是否上线。示例 `THREAD_UUID`、`FILE_UUID`、`ARTIFACT_UUID`、`VERSION` 必须换成服务实际返回值；知识库用正整数 ID，文件/会话/产出用真实 UUID。

## 知识库：查询与网页创建

```bash
ccneed kb list
ccneed kb get --kb-id 7
ccneed kb stats --kb-id 7
ccneed kb create
```

`list/get` 显示归属当前用户的采集状态、已发布数据和可选择性。绑定前检查 `can_select`；不可选时说明 `disabled_reason` 或采集状态，不自行猜测可用。

`stats` 只返回已发布、可分析内容的总数、平台与情绪分布、最多八个主话题，以及同环境的 `web_url`。它不返回帖子正文、单条来源或批量数据。用于选库和判断覆盖范围；如果用户要研究结论，仍需 `ccneed run ... --kb-id 7` 交给 CCNeed 云端分析。需要在网页浏览具体内容时，把 `web_url` 交给用户，不通过其他网页接口绕过 CLI 的聚合边界。采集中的数据尚未发布时，统计可能暂时为零，不把零解释为采集失败。

`kb create` 只返回 `status: website_required` 和当前 origin 的官网入口 `url`，**没有创建知识库或启动采集**。向用户展示链接，把意图优化、平台/配额选择和确认交给网页；完成后重新 `kb list/get` 获取真实 ID 和状态，不伪造知识库。

## 文件：先上传，再绑定

```bash
ccneed files list --limit 200
ccneed files upload ./research-brief.pdf
```

列表 `--limit` 范围 1–500，关注 `has_more_possible`，不宣称完整历史。上传只接受非空普通支持文件，最大 100 MiB，不跟随符号链接；支持 csv、txt、md、json、ipynb、pdf、docx、xlsx 及 png/jpg/jpeg/webp/gif。

这里上传的是用户自己的文件。成功返回 `library_file_id`、`display_name`、`size_bytes`、`status`，**尚未绑定会话或启动任务**。接收未知先查 `files list`/网站核对，不自动重传制造重复资料。

开启会话时用第一句话绑定已有资源：

```bash
ccneed run "分析这些资料并生成报告" --kb-id 7 --file-id FILE_UUID
```

`--kb-id` 和 `--file-id` 可重复或逗号分隔；不要把本地文件路径传给 `--file-id`。先确认用户要分析的资料，不按同名文件自动绑定。

## 追加绑定：受窗口与版本约束

```bash
ccneed threads resources --thread-id THREAD_UUID
ccneed threads append --thread-id THREAD_UUID --file-id FILE_UUID --expected-version VERSION
```

读取返回 `kb_ids`、`library_file_ids`、`metadata_version`、`resource_window_state`。追加时沿用原样 `metadata_version`（不是随意编造的整数），只在服务器允许的 `report_window` 操作；也可追加 `--kb-id`。

只支持增加已有资源，不支持解绑、替换或运行中任意修改。窗口关闭或版本冲突时重新读取并与用户对齐，不改参数绕过，也不静默使用旧版本重试。

## findings/reports：列表、文本、原文件下载与 PDF 导出

```bash
ccneed artifacts list --thread-id THREAD_UUID
ccneed artifacts read --thread-id THREAD_UUID --artifact-id ARTIFACT_UUID
ccneed artifacts download --thread-id THREAD_UUID --artifact-id ARTIFACT_UUID --output ./report.md
ccneed artifacts export-pdf --thread-id THREAD_UUID --artifact-id REPORT_UUID --output ./report.pdf
```

列表仅暴露网站可见的 findings/reports，`has_more` 为真不能声称列出了全部。需要**本轮产出**时，以 `tasks get` 的关联 `artifacts` 为准，不能把会话目录里的所有旧文件都归给本轮。

`read` 返回 JSON 文本 `text`；二进制为 `binary_artifact`，应在用户需要文件时使用显式下载，不直接打印二进制。读取和下载都受当前 5 MiB 上限约束，超限明确报错。

用户说“看完整报告”“发我报告”或关心图表、分析呈现和排版时，优先对 `report` 类型使用 `export-pdf`：它调用同一环境 Club 的 PDF 渲染服务，保存真正的 `.pdf`。用户明确要原始内容、继续编辑或机器处理时，才用 `download` 保存原始 artifact 字节；Markdown 报告用 `.md` 后缀。`export-pdf` 不能用于 findings，也不能把 `.md` 改后缀冒充 PDF。服务端尚未部署此接口时如实说明，别把原文件下载说成导出成功。

下载必须指定本地目标；不采用服务器文件名、不覆盖已有文件。现有目标冲突时选新名字或询问用户，不删除旧文件来绕过保护。只有完整下载安全发布后才能声称文件已保存。

读取/下载是当前工作空间字节，不是不可变历史快照。没有公开产出时如实说明，不扫描隐藏路径、对象存储地址或任意沙箱文件。
