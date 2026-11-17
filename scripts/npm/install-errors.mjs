const permissionHint = 'Write access was denied. Check the npm cache, global installation directory or host files named in the local log. Retry from a terminal or Agent session with permission to write that location; a fresh cache does not grant global write access.';
const hints = {
  EPERM: permissionHint,
  EACCES: permissionHint,
  EBUSY: 'A cache or installation file is in use. Close the process using that file and retry. If the local npm log identifies the cache, retry with a fresh npm cache directory; this does not fix global directory permissions.',
  ENOENT: 'A required executable or file was not found. Check Node.js, npm and the selected host installation and PATH, then retry.',
  ETIMEDOUT: 'The operation timed out. Check registry access, proxy settings or the selected host, then retry the installer.',
  ENOTFOUND: 'A server name could not be resolved. Check network access, registry and proxy settings, then retry.',
  EAI_AGAIN: 'Name resolution temporarily failed. Check network access and retry when it is available.',
  ECONNRESET: 'The connection was interrupted. Check network and proxy settings, then retry.',
  MARKETPLACE_UNAVAILABLE: 'The native host could not load its marketplaces. Inspect its local marketplace list command and repair the reported source, then retry; CLI installation and plugin registration are separate stages.',
  INTERRUPTED: 'The command was interrupted or timed out. Check the selected tool and retry when it is available.',
  COMMAND_FAILED: 'Inspect the local npm log or selected host command for details, then retry. Raw command output is withheld because it may contain credentials.',
};

/** installationError preserves a bounded failure category without exposing child output.
 * Args:
 *   error: A filesystem or child-process error; raw messages and output stay private.
 *   stage: Installer-controlled label identifying the failed operation.
 * Returns:
 *   An Error with safe code, stage and optional numeric exitCode for user-facing output.
 */
export function installationError(error, stage) {
  const stderr = typeof error?.stderr === 'string' ? error.stderr.slice(0, 1024 * 1024) : '';
  const npmCode = /^npm (?:error|ERR!) code ([A-Z][A-Z0-9_]*)\s*$/m.exec(stderr)?.[1];
  let code = Object.hasOwn(hints, error?.code) ? error.code : Object.hasOwn(hints, npmCode) ? npmCode : 'COMMAND_FAILED';
  if (code === 'COMMAND_FAILED' && /failed to load marketplace\(s\)/i.test(stderr)) code = 'MARKETPLACE_UNAVAILABLE';
  if (code === 'COMMAND_FAILED' && error?.killed) code = 'INTERRUPTED';
  const exitCode = Number.isInteger(error?.code) ? error.code : error?.exitCode;
  const result = new Error(stage + ' failed [' + code + ']' + (Number.isInteger(exitCode) && exitCode >= 0 && exitCode <= 255 ? ' (exit ' + exitCode + ')' : '') + '. ' + hints[code]);
  result.code = code;
  result.stage = stage;
  if (Number.isInteger(exitCode) && exitCode >= 0 && exitCode <= 255) result.exitCode = exitCode;
  return result;
}
