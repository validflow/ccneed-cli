import assert from 'node:assert/strict';
import test from 'node:test';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

test('native synchronous plugin loaders can import shared guards without executing stdin entry', () => {
  const filename = fileURLToPath(new URL('../../skills/ccneed/scripts/acquisition.mjs', import.meta.url));
  const result = spawnSync(process.execPath, ['-e', 'const hooks=require(process.argv[2]);process.stdout.write(typeof hooks.handleHook)', 'native-loader', filename], { encoding: 'utf8', timeout: 10000, windowsHide: true });
  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.stdout, 'function');
  assert.equal(result.stderr, '');
});
