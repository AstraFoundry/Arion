// Mobile app workflows: generate Sunbeam tokens and the string catalogs,
// check them for drift, and build and test the iOS and Android apps.
//   node scripts/mobile.mjs generate | check | ios-test | android-test
import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const command = process.argv[2];

function run(program, args, cwd = root) {
  const result = spawnSync(program, args, { cwd, stdio: 'inherit' });
  if (result.error) {
    console.error(`${program}: ${result.error.message}`);
    process.exit(1);
  }
  if (result.status !== 0) process.exit(result.status ?? 1);
}

const desktop = path.join(root, 'apps/desktop');
const ios = path.join(root, 'apps/ios');
const android = path.join(root, 'apps/android');
// The simulator the iOS checks run on; override with ARION_IOS_SIMULATOR.
const simulator = process.env.ARION_IOS_SIMULATOR ?? 'iPhone 17 Pro';

switch (command) {
  case 'generate':
    run('go', ['run', './cmd/arion-mobile-tokens'], desktop);
    run('node', ['scripts/generate-mobile-strings.mjs']);
    run('node', ['scripts/generate-mobile-icons.mjs']);
    break;
  case 'check':
    run('go', ['run', './cmd/arion-mobile-tokens', '-check'], desktop);
    run('node', ['scripts/generate-mobile-strings.mjs', '--check']);
    run('node', ['scripts/generate-mobile-icons.mjs', '--check']);
    break;
  case 'ios-test':
    run('xcodegen', ['generate', '--quiet'], ios);
    run('xcodebuild', ['-project', 'Arion.xcodeproj', '-scheme', 'Arion', '-destination', `platform=iOS Simulator,name=${simulator}`,
      '-only-testing:ArionKitTests', 'test', '-quiet'], ios);
    break;
  case 'android-test':
    run(path.join(android, process.platform === 'win32' ? 'gradlew.bat' : 'gradlew'), [':core:test', ':app:assembleDebug', '--quiet'], android);
    break;
  default:
    console.error('usage: node scripts/mobile.mjs generate | check | ios-test | android-test');
    process.exit(2);
}
