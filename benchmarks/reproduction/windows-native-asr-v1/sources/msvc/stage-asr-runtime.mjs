// Stage native ASR code only. Model weights are provisioned separately.
import { execFileSync } from 'node:child_process';
import { cpSync, copyFileSync, existsSync, mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { pinnedFile } from './lib/pinned-file.mjs';
import { verifyWindowsASRPackage } from './lib/windows-pe.mjs';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const platform = process.argv[2] ?? `${process.platform === 'win32' ? 'windows' : process.platform}-${process.arch === 'x64' ? 'amd64' : process.arch}`;
const pins = JSON.parse(readFileSync(path.join(root, 'scripts/asr-runtime.json')));
const work = path.join(root, '.workspace/native-asr-migration');
const resources = path.join(root, 'apps/desktop/resources', platform);
const destination = path.join(resources, 'asr');
const run = (cmd, args, cwd = root) => execFileSync(cmd, args, { cwd, stdio: 'inherit' });
mkdirSync(work, { recursive: true });
rmSync(destination, { recursive: true, force: true });
mkdirSync(path.join(destination, 'licenses'), { recursive: true });
// Remove obsolete generated runtime artifacts from incremental builds.
rmSync(path.join(resources, 'lib'), { recursive: true, force: true });
if (existsSync(resources)) for (const name of readdirSync(resources)) if (/sherpa|onnxruntime/i.test(name)) rmSync(path.join(resources, name), { recursive: true, force: true });
if (platform === 'darwin-arm64') {
  const source = path.join(root, 'apps/desktop/native/mlx-asr');
  const derived = path.join(work, 'build');
  run('xcodebuild', ['-quiet', '-scheme', 'ArionASR', '-configuration', 'Release', '-destination', 'platform=macOS,arch=arm64', '-derivedDataPath', derived, '-onlyUsePackageVersionsFromResolvedFile', 'MACOSX_DEPLOYMENT_TARGET=14.0', 'build'], source);
  const products = path.join(derived, 'Build/Products/Release');
  copyFileSync(path.join(products, 'arion-asr'), path.join(destination, 'arion-asr'));
  const swiftLibraries = path.join(destination, 'lib');
  mkdirSync(swiftLibraries);
  run('xcrun', ['swift-stdlib-tool', '--copy', '--scan-executable', path.join(destination, 'arion-asr'), '--destination', swiftLibraries, '--platform', 'macosx']);
  run('install_name_tool', ['-add_rpath', '@executable_path/lib', path.join(destination, 'arion-asr')]);

  // MLX searches for a colocated mlx.metallib before SwiftPM bundle paths.
  copyFileSync(path.join(products, 'mlx-swift_Cmlx.bundle/Contents/Resources/default.metallib'), path.join(destination, 'mlx.metallib'));
  for (const name of readdirSync(products).filter(n => n.endsWith('.bundle'))) cpSync(path.join(products, name), path.join(destination, name), { recursive: true });
  const checkouts = path.join(derived, 'SourcePackages/checkouts');
  for (const name of readdirSync(checkouts)) {
    const source = path.join(checkouts, name);
    const notices = readdirSync(source).filter(n => /^(LICENSE|LICENCE|COPYING|NOTICE)(\..*)?$/i.test(n));
    if (!notices.length) throw new Error(`Missing Swift dependency license: ${name}`);
    mkdirSync(path.join(destination, 'licenses', name));
    for (const notice of notices) copyFileSync(path.join(source, notice), path.join(destination, 'licenses', name, notice));
  }
  copyFileSync(path.join(source, 'Package.resolved'), path.join(destination, 'Package.resolved'));
} else if (['darwin-amd64', 'windows-amd64'].includes(platform)) {
  const source = path.join(work, 'speech.cpp');
  if (!existsSync(source)) run('git', ['clone', '--no-checkout', pins.speech.repository, source]);
  run('git', ['checkout', '--detach', pins.speech.revision], source);
  run('git', ['submodule', 'update', '--init', '--recursive'], source);
  const dirty = execFileSync('git', ['status', '--porcelain', '--untracked-files=no'], {cwd:source,encoding:'utf8'}).trim();
  if (dirty) throw new Error('Pinned speech.cpp source or submodule has local modifications');

  const build = path.join(work, `speech-${platform}`);
  const flags = ['-S', source, '-B', build, '-DCMAKE_BUILD_TYPE=Release', '-DGGML_NATIVE=OFF', '-DGGML_METAL=OFF', '-DGGML_VULKAN=OFF', '-DGGML_AVX2=OFF', '-DGGML_FMA=OFF', '-DGGML_F16C=OFF', '-DCMAKE_MSVC_RUNTIME_LIBRARY=MultiThreaded'];
  if (platform.startsWith('darwin')) flags.push('-DCMAKE_OSX_ARCHITECTURES=x86_64', '-DCMAKE_OSX_DEPLOYMENT_TARGET=14.0');
  // MSVC /MT does not statically link OpenMP's VCOMP140.DLL. The CPU worker
  // uses ggml's own thread pool and must run without a compiler redistributable.
  if (platform === 'windows-amd64') flags.push('-DGGML_OPENMP=OFF');
  run('cmake', flags);
  run('cmake', ['--build', build, '--config', 'Release', '--target', 'speech-cli', '--parallel', '6']);
  const executable = platform.startsWith('windows') ? 'speech.exe' : 'speech';
  const binary = [path.join(build, executable), path.join(build, 'Release', executable)].find(existsSync);
  if (!binary) throw new Error('Native ASR CPU build did not produce speech');
  copyFileSync(binary, path.join(destination, executable));
  for (const [name, dir] of [['speech.cpp', source], ['ggml', path.join(source, 'ggml')]]) {
    mkdirSync(path.join(destination, 'licenses', name));
    copyFileSync(path.join(dir, 'LICENSE'), path.join(destination, 'licenses', name, 'LICENSE'));
  }
  for (const name of readdirSync(path.join(source, 'vendor'))) {
    const license = path.join(source, 'vendor', name, 'LICENSE');
    if (!existsSync(license)) throw new Error(`Missing bundled ASR vendor license: ${name}`);
    mkdirSync(path.join(destination, 'licenses', name));
    copyFileSync(license, path.join(destination, 'licenses', name, 'LICENSE'));
  }
  if (platform === 'windows-amd64') {
    const archive = pinnedFile(pins.speech.windowsVulkan);
    const extracted = path.join(work, 'speech-vulkan');
    rmSync(extracted, { recursive: true, force: true }); mkdirSync(extracted);
    run('tar', ['-xf', archive, '-C', extracted]);
    const find = dir => readdirSync(dir, { withFileTypes: true }).flatMap(e => e.isDirectory() ? find(path.join(dir,e.name)) : [path.join(dir,e.name)]);
    const executable = find(extracted).find(f => path.basename(f) === 'speech.exe');
    if (!executable) throw new Error('Missing pinned Vulkan executable');
    copyFileSync(executable, path.join(destination, 'speech-vulkan.exe'));
    const checked = verifyWindowsASRPackage(destination, pins.speech.version);
    console.log(`Verified Windows ASR DLL closure and isolated CPU launch: ${checked.version}.`);
  }
} else throw new Error(`Unsupported native ASR platform: ${platform}`);
writeFileSync(path.join(destination, 'runtime.json'), JSON.stringify({ platform, ...pins }, null, 2) + '\n');
console.log(`Staged native ASR for ${platform}.`);
