# Releasing CCNeed CLI

English · [中文](RELEASING.zh.md)

The CLI, npm package, and companion Skill share one version. Maintainers decide when and what to release. Ordinary feature PRs can accumulate on `main`; **do not bump `package.json.version` until a release is intended**. Version numbers are not derived from PR titles, branch names, commit counts, or dates.

Only a version-change PR Squash-merged into `main` initiates the automated release. The Action first validates, tests, and builds six platform assets plus the same-version npm/Skill package without publishing credentials. Only after those pass does a separate job tag, publish the GitHub Release and npm package, and verify the npm channel. A Git tag alone does not mean npm publishing succeeded.

## Current state and prerequisites

The initial stable `0.1.0` and subsequent releases have shipped. Do not repeat the first-package manual bootstrap. Use the existing [Release workflow](.github/workflows/release.yml) and keep repository variable `CCNEED_RELEASE_ENABLED=true`. npm Trusted Publisher must match `validflow/ccneed-cli`, workflow filename `release.yml`, and environment `npm-release`, with `npm publish` and `npm dist-tag` allowed. Verify OIDC on each automated release; no long-lived npm token belongs in the repository or workflow logs.

Before release, confirm:

1. Required `quality` is green; automated CX Review is currently paused. Maintainers review the version change, changelog, and risk.
2. `package.json`, root `package-lock.json` package/version, and `CHANGELOG.md` agree. Skill and platform assets ship at that same version.
3. Authorized developers have run real staging integration using a staging account. If Club production lacks a related API, say so in the release notes; publishing the CLI does not deploy Club. Users still connect to `https://ccneed.ai` by default.
4. The `npm-release` environment allows only `main` and does not require a manual reviewer for every release. Keep the repository public so users can download binary assets.

## Publish a version

Create a release branch from current `main`; replace the example with the maintainer-selected **new** version. Never edit `main` directly or tag manually:

```bash
npm version 0.1.4 --no-git-tag-version --ignore-scripts
# Update CHANGELOG.md and CHANGELOG.zh.md, then check:
npm test
bash scripts/check.sh
```

After `quality` passes and the maintainer Squash-merges the PR, the Release Action repeats release checks on the merged commit, then tags, publishes, and verifies. Do not merge the next version PR until the previous release is complete or explicitly failed.

`package.json.version` is the only version source. Stable releases use Git tag `vX.Y.Z` and npm `latest`; prereleases use `vX.Y.Z-rc.N` and npm `next`. `latest` and `next` are movable npm channels, not Git tags. The Action explicitly selects the channel with `npm publish --tag`; an RC suffix alone does not set it. RCs are optional: use them when trial installation is needed, incrementing `N` for fixes before final `X.Y.Z`.

An unchanged version in an ordinary merge does **not** re-tag or republish. A changed `package.json` with an unsynchronized lockfile fails preflight. No weekly schedule forces releases. Breaking changes require maintainer approval and migration notes.

The workflow produces Linux, macOS, and Windows assets for amd64/arm64. The npm installer downloads only its matching exact-version binary and checks SHA-256; users do not need Go. The package contains the installer and full Chinese Skill. The release checksum covers every Skill file, and the installed CLI's actual version is checked. A directory link alone does not prove the host loaded the Skill.

## Failures and safe retries

- Missing version, license, Skill, lockfile, or passing gate stops **before** tagging; test/build jobs have no publishing identity.
- An existing tag can be reused only if it points to the same `main` commit; never move a published tag.
- A draft Release may receive only missing assets after checking every existing asset's size and SHA-256. Stop if a public Release is incomplete; do not silently change its bytes.
- An existing npm version can be reused only if the tarball integrity matches exactly. Never overwrite a version or roll back the stable channel.
- Registry indexes may briefly return 404 after publish. The job checks the version, dist-tag, and tarball integrity every 10 seconds, up to 19 times; different content fails immediately, and a prolonged absence fails without republishing or moving the tag.
- Build artifacts last 30 days. Rerunning only a failed publish job may change `run_attempt` in asset names; rerun the whole workflow instead. Rerunning an old run uses its old commit, not later script fixes.
- If scripts were fixed but `package.json.version` is unchanged, manually dispatch the Release workflow on current `main` with `retry_version` equal to its exact version. It rebuilds and safely resumes that release; ordinary merges never retry it. Conflicting unpublished tags/drafts must first be audited and cleaned up; never move an already successful release.
- The workflow uploads by the draft's returned Release ID, not by assuming tag lookup is immediately consistent. On retries it checks tag, target commit, and asset digests. A public GitHub Release without npm completion is **not** success; fix access/network and resume before announcing.

Success requires verified GitHub assets, npm tarball content, and the intended npm channel. Update checks read stable npm `latest` with a 24-hour cache and two-second request limit; failed checks do not block commands. Updates are advisory, never automatic. Re-run the unified installation command to update CLI and linked Skills together; reload the host. See [Installation](INSTALL.md).

## Stable-release group notification

Only after the publishing job verifies the Git tag, GitHub Release, npm package, and `latest` does the separate `notify` job call the Multica Autopilot using the repository secret `MULTICA_CLI_RELEASE_WEBHOOK_URL`. The stable version has a fixed idempotency key `ccneed-cli:vX.Y.Z`; RCs and unchanged-version merges do not notify the production group. Treat the webhook URL as a credential.

The Autopilot rechecks public release state, then sends the PR, tag, npm version/channel, and link to the group with native @all. A webhook `accepted` response means only that Multica accepted the request; check its run record for actual message delivery. If notification fails after publishing succeeds, do not retag or republish—fix and rerun only the notification.
