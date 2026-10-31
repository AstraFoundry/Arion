import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { changedFiles, nativeShellRequired } from './ci-native-scope.mjs';

const passed = result => result === 'success';
const absent = result => result === 'skipped' || result === 'success';

// A skipped required job does not fail a GitHub Actions workflow by itself.
// This gate fails the run when a check that must run did not succeed.
export function evaluateRequiredChecks({
  event, portable, scope, native, nativeOutput = '', windowsUpdater, windowsRequired,
}) {
  const failures = [];
  if (!passed(portable)) failures.push('portable checks did not succeed');
  if (event === 'pull_request') {
    if (!passed(scope)) failures.push('pull request scope did not succeed');
    if (nativeOutput !== 'true' && nativeOutput !== 'false') failures.push('scope native output must be true or false');
    if (nativeOutput === 'true' && !passed(native)) failures.push('native shell was required and did not succeed');
    if (nativeOutput === 'false' && !absent(native)) failures.push('documentation-only native result must be skipped');
  } else {
    if (scope !== 'skipped') failures.push('scope runs only for pull requests');
    if (!passed(native)) failures.push('main and release runs must succeed at the native shell');
  }
  const nativeRuns = event !== 'pull_request' || nativeOutput === 'true';
  if (windowsRequired && nativeRuns && !passed(windowsUpdater)) failures.push('Windows updater check did not succeed');
  if ((!windowsRequired || !nativeRuns) && !absent(windowsUpdater)) failures.push('Windows updater must not fail when it is not required');
  return failures;
}

export function documentationScopeMismatch(files) {
  return nativeShellRequired(files);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const input = {
    event: process.env.EVENT,
    portable: process.env.PORTABLE,
    scope: process.env.SCOPE,
    native: process.env.NATIVE,
    nativeOutput: process.env.NATIVE_OUTPUT ?? '',
    windowsUpdater: process.env.WINDOWS_UPDATER,
    windowsRequired: process.env.WINDOWS_REQUIRED === 'true',
  };
  const failures = evaluateRequiredChecks(input);
  if (input.event === 'pull_request' && input.nativeOutput === 'false') {
    const files = changedFiles(process.env.BASE_SHA);
    console.log(`ci-scope recheck: ${files.length} files`);
    if (documentationScopeMismatch(files)) failures.push('scope reported documentation-only, but the pull request diff is not');
  }
  if (failures.length) {
    for (const failure of failures) console.error(`ci-required: ${failure}`);
    process.exit(1);
  }
  console.log('ci-required: passed');
}
