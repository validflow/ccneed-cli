#!/usr/bin/env node
import { spawn } from 'node:child_process';
import { installBinary, packageRoot } from './install.mjs';
import { validateBundledSkill } from './skills.mjs';
import { installerMain } from './setup.mjs';

// npm 负责安装/更新配套 Skill 和启动；业务命令全部交给 Go，不经过 shell。
try {
  if (process.argv[2] === 'install') {
    process.stdout.write(await installerMain(process.argv.slice(3)));
  } else {
    validateBundledSkill(packageRoot);
    const binary = await installBinary();
    const child = spawn(binary, process.argv.slice(2), { stdio: 'inherit', windowsHide: true });
    child.on('error', () => { console.error('Unable to start the verified CCNeed executable.'); process.exitCode = 1; });
    child.on('exit', (code, signal) => { process.exitCode = code ?? (signal === 'SIGINT' ? 130 : 1); });
  }
} catch (error) {
  console.error(process.argv[2] === 'install' ? 'CCNeed installation incomplete: ' + error.message : 'CCNeed executable is unavailable. Reinstall the official npm package; do not download an unverified binary.');
  process.exitCode = 1;
}
