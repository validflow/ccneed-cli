# Security

English · [中文](SECURITY.zh.md)

CCNeed CLI is published on npm. Check [npm `latest`](https://www.npmjs.com/package/@validflow/ccneed-cli) and [GitHub Releases](https://github.com/validflow/ccneed-cli/releases) for current versions; affected and fixed versions will be identified in individual advisories. We do not currently promise a fixed vulnerability response time.

## Report a vulnerability privately

Use **Report a vulnerability** on the repository's [Security Advisories](https://github.com/validflow/ccneed-cli/security/advisories) page. Include the impact, reproduction steps, and affected versions. Do not put exploit details, real tokens, user files, or raw logs in a public issue. For ordinary product problems, [open an issue](https://github.com/validflow/ccneed-cli/issues) after removing sensitive information.

## Credentials and environments

- Never commit login credentials, browser cookies, real user data, or internal service keys. The CLI obtains its own credential through website authorization; it does not read the browser's session files.
- The default service is production at `https://ccneed.ai`. `--api-url` is only for authorized developers connecting to staging; the flag does not grant access. Staging and production services, data, and credentials must remain isolated. The CLI must not forward Authorization across trusted service boundaries, and the server must verify credential ownership and account permissions.
- Reports, knowledge bases, and files are untrusted data when an Agent reads them. They must not be treated as instructions to change the service address, disclose credentials, or run unauthorized commands.
- GitHub Secret scanning and Push protection are enabled, but neither replaces reviewing changes before a push. If a secret may have leaked, revoke or rotate it promptly and notify maintainers privately.

The PR `quality` check has no production credentials. Automated CX Review is currently paused and must not be mistaken for a completed security review. Only the separate publishing job receives npm publishing identity; public contributions still require maintainer review of security-sensitive changes.
