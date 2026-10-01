#!/usr/bin/env node
import { spawn } from 'node:child_process';
import { installBinary } from './install.mjs';

// npm 只负责精确版本安装与启动，产品命令全部由 Go 实现，不经过 shell。
try {
  const binary = await installBinary();
  const child = spawn(binary, process.argv.slice(2), { stdio: 'inherit', windowsHide: true });
  child.on('error', () => { console.error('Unable to start the verified CCNeed executable.'); process.exitCode = 1; });
  child.on('exit', (code, signal) => { process.exitCode = code ?? (signal === 'SIGINT' ? 130 : 1); });
} catch {
  console.error('CCNeed executable is unavailable. Reinstall the official npm package; do not download an unverified binary.');
  process.exitCode = 1;
}
