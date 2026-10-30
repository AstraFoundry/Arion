// Stage the pinned offline GGUF runtime and pipe worker for the selected platform.
import { execFileSync } from 'node:child_process';
import { copyFileSync, existsSync, mkdirSync, readFileSync, rmSync, chmodSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { pinnedFile } from './lib/pinned-file.mjs';
import { verifyWindowsLLMPackage } from './lib/windows-pe.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktop = path.join(root, 'apps', 'desktop');
const pins = JSON.parse(readFileSync(path.join(root, 'scripts', 'llm-runtime.json'), 'utf8'));
const run = (command, args, options = {}) => execFileSync(command, args, { stdio: 'inherit', ...options });

const platform = process.argv[2] ?? `${process.platform === 'win32' ? 'windows' : process.platform}-${process.arch === 'x64' ? 'amd64' : process.arch}`;
const spec = pins.platforms[platform];
if (!spec) {
  // No pinned runtime: the app reports the capability as unsupported.
  console.log(`No rewrite runtime is pinned for ${platform}; skipped.`);
  process.exit(0);
}
const destination = path.join(desktop, 'resources', platform, 'llm');
rmSync(destination, { recursive: true, force: true });
mkdirSync(destination, { recursive: true });
mkdirSync(path.join(destination, 'licenses'), { recursive: true });
if (platform === 'windows-amd64') {
  const archive = pinnedFile(spec.source);
  const work = path.join(root, '.workspace', 'local-rewrite');
  const source = path.join(work, 'llama.cpp');
  const build = path.join(work, platform);
  const sourceIdentity = path.join(source, '.arion-source-sha256');
  if (!existsSync(sourceIdentity) || readFileSync(sourceIdentity, 'utf8').trim() !== spec.source.sha256) {
    rmSync(source, { recursive: true, force: true });
    rmSync(path.join(work, platform), { recursive: true, force: true });
    mkdirSync(source, { recursive: true });
    run('tar', ['-xzf', archive, '-C', source, '--strip-components=1']);
    writeFileSync(sourceIdentity, `${spec.source.sha256}\n`);
  }
  // Do not depend on build-host VC/OpenMP redistributables or GPU drivers.
  // MSVC uses /MT; MinGW links its compiler and pthread runtimes statically.
  const flags = ['-S', source, '-B', build, '-DCMAKE_BUILD_TYPE=Release', '-DBUILD_SHARED_LIBS=OFF', '-DGGML_STATIC=ON',
    `-DCMAKE_PROJECT_INCLUDE=${path.join(root, 'scripts', 'cmake', 'windows-llm.cmake')}`,
    '-DGGML_NATIVE=OFF', '-DGGML_OPENMP=OFF', '-DGGML_BACKEND_DL=OFF', '-DGGML_CPU_ALL_VARIANTS=OFF',
    '-DGGML_AVX512=OFF', '-DGGML_AVX_VNNI=OFF', '-DGGML_AVX512_VBMI=OFF', '-DGGML_AVX512_VNNI=OFF', '-DGGML_AVX512_BF16=OFF',
    '-DGGML_CUDA=OFF', '-DGGML_VULKAN=OFF', '-DGGML_SYCL=OFF', '-DGGML_OPENCL=OFF', '-DGGML_HIP=OFF', '-DGGML_RPC=OFF', '-DGGML_BLAS=OFF',
    '-DLLAMA_BUILD_TESTS=OFF', '-DLLAMA_BUILD_EXAMPLES=OFF', '-DLLAMA_OPENSSL=OFF', '-DLLAMA_BUILD_SERVER=ON', '-DLLAMA_BUILD_UI=OFF', '-DLLAMA_USE_PREBUILT_UI=OFF',
    '-DLLAMA_BUILD_NUMBER=11734', `-DLLAMA_BUILD_COMMIT=${spec.source.revision}`, '-DCMAKE_MSVC_RUNTIME_LIBRARY=MultiThreaded'];
  // Compile the same source twice, first the x64 baseline and then the bounded
  // AVX2 feature set checked by the worker before it chooses that executable.
  for (const [name, enabled] of [['llama-server.exe', 'OFF'], ['llama-server-avx2.exe', 'ON']]) {
    run('cmake', [...flags, ...['SSE42', 'AVX', 'AVX2', 'FMA', 'F16C', 'BMI2'].map(feature => `-DGGML_${feature}=${enabled}`)]);
    run('cmake', ['--build', build, '--config', 'Release', '--target', 'llama-server', '--parallel', '6']);
    const binary = [path.join(build, 'bin', 'llama-server.exe'), path.join(build, 'bin', 'Release', 'llama-server.exe')].find(existsSync);
    if (!binary) throw new Error(`Pinned Windows GGUF build did not produce ${name}`);
    copyFileSync(binary, path.join(destination, name));
  }
  copyFileSync(path.join(source, 'LICENSE'), path.join(destination, 'licenses', 'llama.cpp-LICENSE'));
  copyFileSync(path.join(source, 'vendor', 'cpp-httplib', 'LICENSE'), path.join(destination, 'licenses', 'cpp-httplib-LICENSE'));
  copyFileSync(path.join(source, 'licenses', 'LICENSE-jsonhpp'), path.join(destination, 'licenses', 'jsonhpp-LICENSE'));
  const checked = verifyWindowsLLMPackage(destination);
  console.log(`Verified Windows GGUF DLL closure and isolated CPU launch: ${checked.version}.`);
}
for (const archive of spec.archives ?? []) {
  const file = pinnedFile(archive);
  const unpacked = `${file}.d`;
  rmSync(unpacked, { recursive: true, force: true });
  mkdirSync(unpacked);
  run('tar', ['-xzf', file, '-C', unpacked, ...Object.keys(archive.members)]);
  for (const [member, name] of Object.entries(archive.members)) {
    const target = path.join(destination, name);
    mkdirSync(path.dirname(target), { recursive: true });
    copyFileSync(path.join(unpacked, member), target);
    chmodSync(target, name.endsWith('.dylib') || name === 'llama-server' ? 0o755 : 0o644);
  }
  rmSync(unpacked, { recursive: true, force: true });
}
for (const name of ['cmudict-pronunciation-v1-LICENSE', 'pypinyin-character-readings-v1-LICENSE.txt', 'cmudict-pronunciation-v1-provenance.json', 'pypinyin-character-readings-v1-provenance.json']) {
  copyFileSync(path.join(desktop, 'infrastructure/llm/spelling/data', name), path.join(destination, 'licenses', name));
}
const [goos, goarch] = platform.split('-');
const worker = path.join(destination, goos === 'windows' ? 'arion-llm.exe' : 'arion-llm');
run('go', ['build', '-trimpath', '-o', worker, './cmd/arion-llm'], { cwd: desktop, env: { ...process.env, CGO_ENABLED: '0', GOOS: goos, GOARCH: goarch } });
console.log(`${path.relative(root, destination)}: ${[...(spec.files ?? []), ...(spec.archives ?? []).flatMap(a => Object.values(a.members))].join(', ')}, ${path.basename(worker)}`);
