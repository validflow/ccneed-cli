# Changelog

English · [中文](CHANGELOG.zh.md)

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
