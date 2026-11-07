import { pathToFileURL } from 'node:url';
import { runHookFromStdin } from '../skills/ccneed/scripts/acquisition.mjs';

export { renderPlan, handleHook } from '../skills/ccneed/scripts/acquisition.mjs';

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const hostIndex = process.argv.indexOf('--host');
  await runHookFromStdin(hostIndex < 0 ? '' : process.argv[hostIndex + 1]);
}
