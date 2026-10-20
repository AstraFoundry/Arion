// Release checks never create keys or weaken update signature verification.
import { execFileSync, spawnSync } from 'node:child_process';
import { copyFileSync, mkdirSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { checkRelease, digest, githubAssetName, platformAssets, platformsForRelease, releaseMatrix, releaseStatus } from './lib/release.mjs';
import { checkOfficialCenter } from './lib/center-profile.mjs';
import { macosSigningSettings } from './lib/desktop-signing.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const config = JSON.parse(readFileSync(path.join(root, 'apps/desktop/desktop.json'), 'utf8'));
const [operation, first, second, mode] = process.argv.slice(2);
const checksumList = (directory, files) => [...files, 'RELEASE-STATUS.md'].sort().map(file => `${digest(readFileSync(path.join(directory, file))).toString('hex')}  ${githubAssetName(file)}\n`).join('');
if (operation === 'matrix') {
  console.log(JSON.stringify(releaseMatrix(config)));
} else if (operation === 'check') {
  checkOfficialCenter({ ...process.env, ARION_OFFICIAL_RELEASE: '1' });
  checkRelease(config, process.env.GITHUB_REF_NAME, process.env.MYGO_UPDATER_PRIVATE_KEY);
  const signing = macosSigningSettings(process.env);
  if (!signing?.certificate || !signing.appleId) throw new Error('Tagged releases require a Developer ID certificate, its password, signing identity and Apple notarization credentials');
  const profile = process.env.MACOS_IME_PROVISIONING_PROFILE_BASE64?.trim();
  if (!profile || !Buffer.from(profile, 'base64').length || Buffer.from(profile, 'base64').toString('base64') !== profile) throw new Error('Tagged releases require MACOS_IME_PROVISIONING_PROFILE_BASE64');
  if (process.env.GITHUB_REPOSITORY !== config.updates.github) throw new Error('Release repository does not match the configured update source');
  console.log(`Verified release configuration for v${config.version}.`);
} else if (operation === 'collect') {
  if (!['--signed', '--unsigned'].includes(mode)) throw new Error('Choose --signed for releases or --unsigned for preview installers');
  const platform = first;
  const destination = path.resolve(second);
  const source = path.join(root, 'apps/desktop', config.out, platform);
  const files = platformAssets(source, config, platform, mode === '--signed');
  mkdirSync(destination, { recursive: true });
  for (const file of files) copyFileSync(path.join(source, file), path.join(destination, file));
  console.log(`Collected ${platform}: ${files.join(', ')}`);
} else if (operation === 'verify' || operation === 'upload') {
  const directory = path.resolve(first);
  const platforms = platformsForRelease(config);
  const files = platforms.flatMap(platform => platformAssets(directory, config, platform));
  if (operation === 'verify') {
    const unexpected = readdirSync(directory).filter(file => !files.includes(file));
    if (unexpected.length) throw new Error(`Unexpected release assets: ${unexpected.join(', ')}`);
    writeFileSync(path.join(directory, 'RELEASE-STATUS.md'), releaseStatus(config, readFileSync(path.join(root, 'docs/operations/release-status.md'), 'utf8')));
    writeFileSync(path.join(directory, 'SHA256SUMS'), checksumList(directory, files));
    console.log(`Verified installers and signed updates for ${platforms.join(', ')}.`);
  } else {
    const tag = `v${config.version}`;
    if (process.env.GITHUB_REF_NAME !== tag || process.env.GITHUB_REPOSITORY !== config.updates.github) throw new Error('Release tag or repository mismatch');
    const allFiles = [...files, 'RELEASE-STATUS.md', 'SHA256SUMS'];
    if (readFileSync(path.join(directory, 'SHA256SUMS'), 'utf8') !== checksumList(directory, files)) throw new Error('Release checksums changed after verification');
    // A failed lookup must not be treated as permission to create a new release.
    const list = execFileSync('gh', ['release', 'list', '--repo', config.updates.github, '--limit', '100', '--json', 'tagName,isDraft'], { encoding: 'utf8' });
    const existing = JSON.parse(list).find(release => release.tagName === tag);
    if (existing && !existing.isDraft) throw new Error('Refusing to modify a published release');
    if (!existing) execFileSync('gh', ['release', 'create', tag, '--repo', config.updates.github, '--draft', '--verify-tag', '--title', tag, '--generate-notes', ...(config.version.includes('-') ? ['--prerelease'] : [])], { stdio: 'inherit' });
    const result = spawnSync('gh', ['release', 'upload', tag, '--repo', config.updates.github, '--clobber', ...allFiles.map(file => path.join(directory, file))], { stdio: 'inherit' });
    if (result.error) throw result.error;
    if (result.status !== 0) process.exit(result.status ?? 1);
  }
} else throw new Error('Use release.mjs matrix | check | collect <platform> <directory> --signed|--unsigned | verify <directory> | upload <directory>');
