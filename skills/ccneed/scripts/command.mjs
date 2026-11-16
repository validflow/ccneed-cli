/** substitutionEnd finds a closing parenthesis without counting quoted or escaped literals.
 * Args:
 *   source: Host shell text.
 *   start: First character inside a command substitution.
 * Returns:
 *   The closing parenthesis position, or the end of an unfinished substitution.
 */
function substitutionEnd(source, start) {
  let quote = '', depth = 1;
  for (let index = start; index < source.length; index++) {
    const char = source[index];
    if (char === '\\' && quote !== "'") { index++; continue; }
    if (quote === "'") { if (char === "'") quote = ''; continue; }
    if (char === '"') { quote = quote ? '' : '"'; continue; }
    if (char === '$' && source[index + 1] === '(') { index = substitutionEnd(source, index + 2); continue; }
    if (quote) continue;
    if (char === '#' && (index === start || /[\s;&|()]/.test(source[index - 1]))) {
      const end = source.indexOf('\n', index);
      if (end < 0) return source.length;
      index = end; continue;
    }
    if (char === "'") { quote = char; continue; }
    if (char === '(') depth++;
    if (char === ')' && --depth === 0) return index;
  }
  return source.length;
}

/** commandGroups separates command positions from literal arguments without executing shell text.
 * Args:
 *   source: A host shell command.
 * Returns:
 *   Command groups and active command substitutions.
 */
function commandGroups(source) {
  const groups = [], nested = [];
  let words = [], word = '', quote = '', started = false;
  const flushWord = () => { if (started) words.push(word); word = ''; started = false; };
  const flushGroup = () => { flushWord(); if (words.length) groups.push(words); words = []; };
  for (let index = 0; index < source.length; index++) {
    const char = source[index], next = source[index + 1];
    if (quote !== "'" && char === '$' && next === '(') {
      const end = substitutionEnd(source, index + 2);
      nested.push(source.slice(index + 2, end));
      word += '<substitution>'; started = true; index = end; continue;
    }
    if (quote !== "'" && char === '`') {
      const end = source.indexOf('`', index + 1);
      nested.push(source.slice(index + 1, end < 0 ? source.length : end));
      word += '<substitution>'; started = true; index = end < 0 ? source.length : end; continue;
    }
    if (char === '\\' && quote !== "'" && next && /[\s"';&|()$`\\]/.test(next)) {
      word += next; started = true; index++; continue;
    }
    if (quote) {
      if (char === quote) quote = '';
      else word += char;
      continue;
    }
    if (char === '"' || char === "'") { quote = char; started = true; continue; }
    if (char === '#' && !started) {
      const end = source.indexOf('\n', index);
      index = end < 0 ? source.length : end - 1; continue;
    }
    if (char === '<' || char === '>') {
      flushWord(); words.push(char);
      if (next === char || next === '&') index++;
      continue;
    }
    if (/[;&|(){}\r\n]/.test(char)) { flushGroup(); continue; }
    if (/\s/.test(char)) { flushWord(); continue; }
    word += char; started = true;
  }
  flushGroup();
  return { groups, nested };
}

/** executableName compares both Windows and POSIX executable paths.
 * Args:
 *   value: A literal command word.
 * Returns:
 *   Its lowercase basename without a platform executable suffix.
 */
function executableName(value) {
  return value.split(/[\\/]/).at(-1).toLowerCase().replace(/\.(?:exe|cmd|bat)$/, '');
}

/** invokesCCNeed follows known execution wrappers rather than searching ordinary argument data.
 * Args:
 *   words: One shell command's literal words.
 *   depth: Current wrapper nesting depth.
 * Returns:
 *   Whether the command invokes CCNeed directly or through an execution wrapper.
 */
function invokesCCNeed(words, depth) {
  if (depth > 32) throw new Error('Command nesting is too deep');
  words = words.filter((word, index) => !['<', '>'].includes(word) && !['<', '>'].includes(words[index - 1]) && !(/^\d+$/.test(word) && ['<', '>'].includes(words[index + 1])));
  let index = 0;
  while (/^[a-z_][\w]*=/i.test(words[index] ?? '') || ['if', 'while', 'until', 'elif', 'then', 'else', 'do', '!'].includes(words[index])) index++;
  if (index >= words.length) return false;
  const name = executableName(words[index]);
  const args = words.slice(index + 1);
  if (name === 'ccneed') return true;
  if (['eval', 'invoke-expression', 'iex', 'call'].includes(name)) return hasCCNeedInvocation(args.join(' '), depth + 1);
  if (['env', 'command', 'exec', 'sudo', 'nohup', 'time', 'timeout', 'nice', 'stdbuf', 'xargs', '.'].includes(name)) {
    let start = 0;
    const valueFlags = { env: ['-u', '--unset', '-C', '--chdir', '--argv0'], sudo: ['-u', '--user', '-g', '--group', '-h', '--host', '-p', '--prompt', '-C', '-D', '--chdir', '-r', '-t', '-T', '--command-timeout'], exec: ['-a'], time: ['-f', '--format', '-o', '--output'], stdbuf: ['-i', '-o', '-e', '--input', '--output', '--error'], timeout: ['-s', '--signal', '-k', '--kill-after'], nice: ['-n', '--adjustment'], xargs: ['-a', '--arg-file', '-d', '--delimiter', '-E', '-I', '-L', '-n', '-P', '-s', '--max-args', '--max-lines', '--max-procs', '--max-chars'] }[name] ?? [];
    const switches = { env: ['-i', '--ignore-environment', '-0', '--null', '-v', '--debug'], sudo: ['-n', '--non-interactive', '-S', '--stdin', '-E', '--preserve-env', '-H', '--set-home', '-b', '--background', '-k', '-K', '-i', '-s'], exec: ['-c', '-l'], time: ['-p', '--portability', '-v', '--verbose', '-a', '--append'], timeout: ['--preserve-status', '--foreground', '-v', '--verbose'], xargs: ['-0', '--null', '-r', '--no-run-if-empty', '-t', '--verbose', '-p', '--interactive', '-x', '--exit'], command: ['-p'] }[name] ?? [];
    while (start < args.length && (args[start].startsWith('-') || /^[a-z_][\w]*=/i.test(args[start]))) {
      if (name === 'command' && ['-v', '-V'].includes(args[start])) return false;
      if (name === 'env' && ['-S', '--split-string'].includes(args[start])) return hasCCNeedInvocation(args.slice(start + 1).join(' '), depth + 1);
      if (name === 'env' && args[start].startsWith('--split-string=')) return hasCCNeedInvocation(args[start].slice(15) + ' ' + args.slice(start + 1).join(' '), depth + 1);
      const flag = args[start];
      if (flag.startsWith('-') && flag !== '--' && !flag.includes('=') && !valueFlags.includes(flag) && !switches.includes(flag)) return args.slice(start + 1).some(value => hasCCNeedInvocation(value, depth + 1));
      start += valueFlags.includes(args[start]) ? 2 : 1;
    }
    if (name === 'timeout') start++;
    return invokesCCNeed(args.slice(start), depth + 1);
  }
  if (name === 'find') {
    return args.some((value, index) => ['-exec', '-execdir', '-ok', '-okdir'].includes(value) && invokesCCNeed(args.slice(index + 1), depth + 1));
  }
  if (/^(?:node|nodejs|python(?:\d+(?:\.\d+)*)?|py)$/.test(name)) {
    const flag = args.findIndex(value => ['-e', '--eval', '-c'].includes(value));
    if (flag < 0) return false;
    const script = args[flag + 1] ?? '';
    const calls = script.matchAll(/\b(?:exec(?:File)?(?:Sync)?|spawn(?:Sync)?|Popen|run|system|popen|call|check_call|check_output)\s*\(\s*(?:\[\s*)?(["'`])((?:\\.|(?!\1)[\s\S])*?)\1/g);
    return [...calls].some(match => hasCCNeedInvocation(match[2].replace(/\\(["'`\\])/g, '$1'), depth + 1));
  }
  if (['bash', 'sh', 'zsh', 'dash', 'fish', 'cmd', 'powershell', 'pwsh'].includes(name)) {
    const flag = args.findIndex(value => /^(?:-[a-z]*c|\/c|-command)$/i.test(value));
    return flag >= 0 && hasCCNeedInvocation(args.slice(flag + 1).join(' '), depth + 1);
  }
  if (['npx', 'npm', 'pnpm', 'yarn', 'bun'].includes(name)) return packageRunnerInvokes(name, args, depth);
  return false;
}

/** packageRunnerInvokes distinguishes the installer, literal executables and runner shell scripts.
 * Args:
 *   name: Package manager executable name.
 *   args: Literal manager arguments.
 *   depth: Current shell-wrapper depth.
 * Returns:
 *   Whether a product command is executed, excluding the unified plugin installer.
 */
function packageRunnerInvokes(name, args, depth) {
  let start = 0, runner = name === 'npx';
  while (start < args.length) {
    const value = args[start];
    if (['-c', '--call'].includes(value)) return hasCCNeedInvocation(args[start + 1], depth + 1);
    if (value.startsWith('--call=')) return hasCCNeedInvocation(value.slice(7), depth + 1);
    if (['--help', '-h', '--version'].includes(value)) return false;
    if (['--package', '-p', '--cache', '--registry', '--filter', '-C', '--dir', '--prefix', '--userconfig', '--globalconfig', '--workspace', '-w', '--loglevel'].includes(value)) { start += 2; continue; }
    if (value.startsWith('-')) {
      if (value !== '--' && !value.includes('=') && !['-y', '--yes', '-s', '--silent', '--offline', '--ignore-scripts', '--no', '--quiet'].includes(value)) return args.slice(start + 1).some((word, index) => hasCCNeedInvocation(word, depth + 1) || (['exec', 'x', 'dlx'].includes(word) && packageRunnerInvokes('npx', args.slice(start + index + 2), depth + 1)));
      start++; continue;
    }
    if (!runner) {
      if (!['exec', 'x', 'dlx'].includes(value)) return false;
      runner = true; start++; continue;
    }
    if (/^@validflow\/ccneed-cli(?:@[^\s]+)?$/.test(value)) return args[start + 1] !== 'install';
    return invokesCCNeed(args.slice(start), depth + 1);
  }
  return false;
}

/** hasCCNeedInvocation identifies CCNeed command nodes before acquisition state is accessed.
 * Args:
 *   source: Host command text; this is never evaluated.
 *   depth: Internal recursion bound for nested shell wrappers.
 * Returns:
 *   Whether CCNeed is invoked, including compound commands and substitutions.
 * Errors:
 *   Excessively nested commands cannot be safely classified.
 */
export function hasCCNeedInvocation(source, depth = 0) {
  if (typeof source !== 'string') return false;
  if (depth > 32) throw new Error('Command nesting is too deep');
  const { groups, nested } = commandGroups(source);
  return groups.some(words => invokesCCNeed(words, depth)) || nested.some(value => hasCCNeedInvocation(value, depth + 1));
}
