import { execFileSync } from 'node:child_process';
import { readFileSync, mkdirSync, copyFileSync, existsSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { git } from './git.mjs';
import { validateManifest } from './version.mjs';
import { targets, stagePackage, verifyArtifacts } from './artifacts.mjs';

// 所有构建身份来自当前 main manifest 和 SHA；使用新输出目录，避免旧产物混入。
try {
  const root = process.cwd();
  const manifest = JSON.parse(readFileSync('package.json', 'utf8'));
  validateManifest(manifest, process.env.RELEASE_TAG);
  const sha = git(['rev-parse', 'HEAD']);
  if (sha !== process.env.GITHUB_SHA) throw new Error('Build revision mismatch');
  const out = resolve('dist');
  if (existsSync(out)) throw new Error('Release requires a fresh dist directory');
  // 在编译之前拒绝缺失的产品 Skill 和许可证，不把开发骨架发布为可用产品。
  readFileSync(join(root, 'skills', 'ccneed', 'SKILL.md'));
  readFileSync(join(root, 'LICENSE'));
  mkdirSync(out);
  const flags = '-s -w -X main.buildVersion=' + manifest.version + ' -X main.buildCommit=' + sha + ' -X main.buildPackage=' + manifest.name;
  for (const { os, arch, asset } of targets(manifest.version)) {
    execFileSync('go', ['build', '-trimpath', '-buildvcs=false', '-ldflags', flags, '-o', join(out, asset), './cmd/ccneed'], {
      env: { ...process.env, CGO_ENABLED: '0', GOOS: os, GOARCH: arch }, stdio: 'inherit', timeout: 180_000,
    });
  }
  const staging = stagePackage(root, out, manifest);
  verifyArtifacts(out);
  const native = targets(manifest.version).find((target) => target.os === 'linux' && target.arch === 'amd64');
  if (process.platform === 'linux' && process.arch === 'x64') {
    const info = JSON.parse(execFileSync(join(out, native.asset), ['version', '--no-update-check'], { encoding: 'utf8', timeout: 10_000 }));
    if (info.version !== manifest.version || info.commit !== sha) throw new Error('Binary build identity mismatch');
  }
  const packed = JSON.parse(execFileSync('npm', ['pack', '--json', '--ignore-scripts'], { cwd: staging, encoding: 'utf8', timeout: 60_000 }));
  if (packed.length !== 1 || !/^[a-zA-Z0-9._-]+\.tgz$/.test(packed[0].filename)) throw new Error('Invalid npm pack result');
  const included = new Set(packed[0].files.map((file) => file.path));
  for (const required of ['checksums.json', 'scripts/npm/install.mjs', 'scripts/npm/run.mjs', 'skills/ccneed/SKILL.md']) if (!included.has(required)) throw new Error('npm package misses required release content');
  copyFileSync(join(staging, packed[0].filename), join(out, 'ccneed-npm-' + manifest.version + '.tgz'));
} catch (error) {
  console.error('Release build failed: ' + error.message);
  process.exitCode = 1;
}
