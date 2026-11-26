# Changelog

English · [中文](../zh/CHANGELOG.md)

## 0.1.18

- Simplify collection confirmation: the Agent presents the `kb plan` result and starts after user confirmation. Cloud HITL replies convey the user's answer.
- Remove extra execution approvals and mode restrictions from native plugins, avoiding CCNeed rule conflicts with Codex Never mode and `kb start --help`.
- Migrate Codex rules that exactly match the previous installer template while preserving custom rules; review legacy WorkBuddy rules manually. Update the shared Skill, installation guidance and plugin versions.

## 0.1.17

- Clear an earlier cloud confirmation when a new task read has no pending action, and restore ZCode startup guidance when a session resumes.
- Clarify Codex's native command approval and Qoder's permission pipeline.

## 0.1.16

- Add optional `ccneed auth login --agent` metadata to display the host that initiated sign-in, such as Codex, Claude Code or WorkBuddy. Omitting it keeps the generic CCNeed CLI authorization flow; the parameter does not change permissions or bind credentials to a host.
- Update the shared Skill and native startup guidance to identify the current executing host. Prefer `--agent` when the native plugin explicitly supplies that identity; never infer it from the model, installed applications or user content. Unknown hosts omit the parameter, and `--complete` keeps the existing proof without repeating it.
- Clarify that agents sharing the same OS user, credential store and service origin can reuse an existing valid CLI sign-in. Switching agents does not require fresh authorization.

Host display and the redesigned browser authorization page require the matching Club migration and deployment before publishing this CLI version. Updating the CLI alone does not deploy the website changes.

## 0.1.15

- Add `ccneed kb platforms` to query the deployment's collection platform IDs. The shared Skill checks this list before preparing create or supplement plans, including newly enabled platforms.
- Explain installation failures with actionable diagnostics for permissions, occupied files, network errors and other npm failures, while redacting credentials.

Platform discovery requires the corresponding Club deployment to allow CLI credentials on the existing platform endpoint.

## 0.1.14

- Add native plugin/extension payloads for OpenClaw, Pi, ZCode, Qoder, Trae CLI and DeepSeek Harness, with host-specific tool events and execution approvals sharing the existing plan and HITL validation.
- Extend the installer with explicit host selection and native manager setup/update steps. Validate complete same-version plugin payloads during packaging.
- Center equal-width Agent icons, show each name once and support dark backgrounds. Keep one install command and the existing ask-your-Agent entry.

## 0.1.13

- Move the Kimi manifest to `.kimi-plugin/plugin.json`, matching the other native plugin directories; update installation validation and release packaging.
- Organize detailed English and Chinese guides under `docs/en/` and `docs/zh/`, keeping the READMEs and GitHub contribution/security entries at the root. Update links, installation prompts and release-note paths; include both languages in the npm package.
- Keep maintainer review configuration local and ignored while preserving the Codex product marketplace. Automatic CI review remains paused.

## 0.1.12

- Fix collection hooks blocking ordinary commands whose titles, search terms or paths contain `ccneed`; executable lookup and the unified installer also pass without touching collection state.
- Preserve plan and native approval checks for actual collection calls, including shell wrappers, compound commands and command substitutions. Report command-format errors separately from plan-validation errors.
- Add a Kimi Code native plugin, with loading and updates verified on 2.5.3. Execution follows the user's native permission mode, including automatic/session authorization, while plan and credit checks remain active.
- Add optional `tasks get/watch --output` snapshots, preserving stdout and preventing overwrites. Kimi hooks read the explicit complete snapshot when the host truncates a long plan or HITL result.
- Add compact Agent support tables with 20px icons to both READMEs, including a dark-theme Kimi icon.

## 0.1.11

- Describe platform quotas as collection targets and explain that complete result pages can exceed them, for direct collection and cloud confirmations across Claude Code, Codex and WorkBuddy.
- Clarify that CLI submissions use the platform's normal collection and tagging workflow; the Agent waits for completion before automatically exporting CSV.
- Simplify the English and Chinese README introductions and capability summaries.

Restoring tagging for CLI collections requires the corresponding Club deployment. This release does not reprocess existing data or change credit accounting.

## 0.1.10

- Clarified the shared Skill and hook context: each plan confirmation includes the complete requirement, platform entity/other keywords and quotas, total count, estimated credits, balance and adjustments, with flexible prose, tables or lists.
- Entity and other keywords can be added, removed or replaced after revalidation. Each platform must retain a nonblank entity; blank-only entity groups now invalidate the prepared approval candidate.
- Kept the short execution reminder, native permission UI and automatic CSV delivery after collection completes. Added regression coverage for repeated reviews and keyword edits across Claude Code, Codex and WorkBuddy.

## 0.1.9

- Collection hooks now pass validated plans and field errors into Agent context. The Agent presents the plan conversationally and waits for user confirmation; execution shows a short reminder and retains native permission approval without repeating the plan.
- Updated the shared Skill to normally export and deliver CSV after collection completes, honoring requested formats and destinations and preserving accurate in-progress or failure states.

## 0.1.8

- Fixed collection hooks rejecting literal Codex plugin-status queries that use `ccneed` as the marketplace name.
- Scoped Codex installation checks to the CCNeed marketplace so unrelated plugin sources cannot delay version verification.

## 0.1.7

CCNeed now includes native Agent plugins and a data collection/export workflow alongside cloud analysis. Export data for your own Agent, or continue research with CCNeed's cloud Agent.

- Added reviewed collection with `kb plan` and `kb start`: validate Agent-generated plans, fit platform quotas to the available balance, display the final plan, and request native execution approval. Unspecified collection counts default to 300 items in total.
- Added knowledge-base CSV/JSON export for analysis by the user's own Agent, with KB identity in default filenames and protection against overwriting existing files.
- Packaged the shared Skill and collection hooks as native Claude Code, Codex and WorkBuddy plugins. The unified installer coordinates CLI and selected native plugin versions, preserves custom configuration and reports incomplete registration.
- Bound collection HITL approvals to the displayed plan, with feedback, rejection and revalidated plan changes supported before cloud execution resumes.

Collection commands require the matching Club APIs to be deployed. WorkBuddy desktop plugin installation requires a native UI step; hooks must be trusted and manual approval must remain enabled.

## 0.1.6

- Fixed GitHub binary downloads during npm installation and recovery on Node.js 22 to honor existing `HTTPS_PROXY`, `HTTP_PROXY`, and `NO_PROXY` settings, including lowercase forms.
- Added actionable proxy download errors without exposing the original network error or credentials; retained trusted redirects, bounded streaming, and SHA-256 verification.
- Updated installation guidance and installed locked npm dependencies without lifecycle scripts before CI tests, builds, and publication.

Website device authorization still depends on the deployed CCNeed service. This CLI update does not deploy the separate website authorization fix.

## 0.1.5

- Added `ccneed kb stats` for aggregate knowledge-base counts and limited distributions, with a link to inspect the knowledge base on the selected website. Post bodies are not returned.
- Added paginated cloud turn progress to `ccneed tasks get`; `tasks watch` now includes the conversation link while preserving task status and confirmation requests.

## 0.1.4

- Shortened the knowledge-base conversation example in both READMEs; detailed CLI workflows remain in the Agent Skill.

## 0.1.3

- Clarified that each new research conversation binds a knowledge base, an uploaded user file, or both; cloud-generated analysis CSVs are not user uploads.
- Made public repository guides English by default, with matching `.zh.md` translations; the bundled Agent Skill remains in Chinese.

## 0.1.2

- Updated English and Chinese README, installation, contribution, release, and security guidance with official social links.
- Clarified that the CLI targets production by default; staging is only for authorized developer integration tests.
- Updated the bundled Skill guidance to match the released product.
- Added a Multica notification after a stable GitHub and npm release is verified; release candidates do not notify the production group.

## 0.1.1

- Fixed the first-release draft lookup failure and allowed safe recovery after interrupted publishing.
- Kept CLI commands and service contracts unchanged.

## 0.1.0

First stable release of CCNeed CLI and its companion Skill through one installation entry point.

- Added website authorization with the existing CCNeed account, credit and account queries.
- Added knowledge-base, file, conversation, and cloud-task management, user confirmations, progress, and history.
- Added Markdown report download and PDF export through the CCNeed service.
- Provided Linux, macOS, and Windows binaries for amd64/arm64.

The CLI can be released independently; some production capabilities depend on Club service deployment. The stable version is installed through npm `latest`.
