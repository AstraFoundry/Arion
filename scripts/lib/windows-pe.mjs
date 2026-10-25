import { readFileSync, readdirSync } from 'node:fs';
import { execFileSync, spawnSync } from 'node:child_process';
import path from 'node:path';

// Windows system components, excluding compiler redistributables. Presence on
// the build host alone does not make a DLL part of a clean Windows install.
const systemLibraries = new Set([
  'advapi32.dll', 'bcrypt.dll', 'comctl32.dll', 'comdlg32.dll', 'crypt32.dll',
  'd2d1.dll', 'd3d11.dll', 'd3d12.dll', 'd3dcompiler_47.dll', 'dcomp.dll',
  'dwrite.dll', 'dxgi.dll', 'gdi32.dll', 'imm32.dll', 'kernel32.dll',
  'msimg32.dll', 'msvcrt.dll', 'ntdll.dll', 'ole32.dll', 'oleaut32.dll',
  'opengl32.dll', 'powrprof.dll', 'propsys.dll', 'psapi.dll', 'rpcrt4.dll',
  'secur32.dll', 'setupapi.dll', 'shell32.dll', 'shcore.dll', 'shlwapi.dll',
  'ucrtbase.dll', 'user32.dll', 'userenv.dll', 'uxtheme.dll', 'version.dll',
  'winhttp.dll', 'winmm.dll', 'ws2_32.dll', 'wtsapi32.dll',
]);

// Read normal and delay-loaded imports directly, without requiring the build
// host's dumpbin/objdump or loading the executable into this process.
export function windowsPEImports(bytes) {
  const requireRange = (offset, size) => {
    if (!Number.isSafeInteger(offset) || offset < 0 || offset + size > bytes.length) throw new Error('Malformed Windows PE range');
  };
  requireRange(0, 64);
  if (bytes.toString('ascii', 0, 2) !== 'MZ') throw new Error('Missing Windows PE DOS header');
  const header = bytes.readUInt32LE(0x3c);
  requireRange(header, 24);
  if (bytes.toString('ascii', header, header + 4) !== 'PE\0\0' || bytes.readUInt16LE(header + 4) !== 0x8664) throw new Error('Expected a Windows x64 PE image');
  const optional = header + 24;
  const optionalSize = bytes.readUInt16LE(header + 20);
  requireRange(optional, optionalSize);
  if (optionalSize < 112 || bytes.readUInt16LE(optional) !== 0x20b) throw new Error('Expected a Windows PE32+ optional header');
  const directoryCount = bytes.readUInt32LE(optional + 108);
  const sectionCount = bytes.readUInt16LE(header + 6);
  const sectionTable = optional + optionalSize;
  requireRange(sectionTable, sectionCount * 40);
  const sections = Array.from({ length: sectionCount }, (_, index) => {
    const section = sectionTable + index * 40;
    return { address: bytes.readUInt32LE(section + 12), size: bytes.readUInt32LE(section + 16), offset: bytes.readUInt32LE(section + 20) };
  });
  const offsetOf = address => {
    if (address < bytes.readUInt32LE(optional + 60)) { requireRange(address, 1); return address; }
    const section = sections.find(section => address >= section.address && address < section.address + section.size);
    if (!section) throw new Error('Windows PE import points outside file-backed sections');
    const offset = section.offset + address - section.address;
    requireRange(offset, 1);
    return offset;
  };
  const nameAt = address => {
    const offset = offsetOf(address);
    const end = bytes.indexOf(0, offset);
    if (end < 0 || end - offset > 4096) throw new Error('Malformed Windows DLL import name');
    const name = bytes.toString('ascii', offset, end).toLowerCase();
    if (!/^[a-z0-9_.-]+\.dll$/.test(name)) throw new Error(`Unsafe Windows DLL import name: ${name}`);
    return name;
  };
  const imports = new Set();
  for (const [index, width, nameOffset] of [[1, 20, 12], [13, 32, 4]]) {
    if (directoryCount <= index) continue;
    if (112 + (index + 1) * 8 > optionalSize) throw new Error('Malformed Windows PE data directory');
    const directory = optional + 112 + index * 8;
    const address = bytes.readUInt32LE(directory), size = bytes.readUInt32LE(directory + 4);
    if (address === 0 && size === 0) continue;
    if (address === 0 || size < width || size > 1024 * 1024) throw new Error('Malformed Windows PE import directory');
    let terminated = false;
    for (let position = 0; position + width <= size; position += width) {
      const offset = offsetOf(address + position);
      requireRange(offset, width);
      if (bytes.subarray(offset, offset + width).every(value => value === 0)) { terminated = true; break; }
      if (index === 13 && bytes.readUInt32LE(offset) !== 1) throw new Error('Windows x64 delay imports must use relative addresses');
      imports.add(nameAt(bytes.readUInt32LE(offset + nameOffset)));
    }
    if (!terminated) throw new Error('Unterminated Windows PE import directory');
  }
  return [...imports];
}

export function verifyWindowsPEDependencies(binaries, { optionalImports = {} } = {}) {
  const pending = [...binaries], verified = new Set();
  while (pending.length) {
    const file = path.resolve(pending.pop());
    if (verified.has(file.toLowerCase())) continue;
    verified.add(file.toLowerCase());
    const allowed = new Set(optionalImports[path.basename(file).toLowerCase()] ?? []);
    const siblings = new Map(readdirSync(path.dirname(file), { withFileTypes: true }).filter(entry => entry.isFile()).map(entry => [entry.name.toLowerCase(), entry.name]));
    for (const library of windowsPEImports(readFileSync(file))) {
      if (systemLibraries.has(library) || /^(api-ms-win-|ext-ms-win-)/.test(library) || allowed.has(library)) continue;
      const bundled = siblings.get(library);
      if (!bundled) throw new Error(`Unbundled Windows DLL dependency: ${file} -> ${library}`);
      pending.push(path.join(path.dirname(file), bundled));
    }
  }
  return [...verified];
}

export function verifyWindowsASRPackage(directory, version) {
  directory = path.resolve(directory);
  const cpu = path.join(directory, 'speech.exe');
  const binaries = [cpu, path.join(directory, 'speech-vulkan.exe')];
  const verified = verifyWindowsPEDependencies(binaries, { optionalImports: { 'speech-vulkan.exe': ['vulkan-1.dll'] } });
  // Exclude compiler/SDK directories from the DLL search path. Vulkan's loader
  // belongs to the optional GPU driver; the CPU worker must launch without it.
  const systemRoot = process.env.SystemRoot;
  if (!systemRoot) throw new Error('Windows ASR smoke requires SystemRoot');
  const output = execFileSync(cpu, ['--version'], {
    cwd: directory, encoding: 'utf8', timeout: 15000, windowsHide: true,
    env: { SystemRoot: systemRoot, WINDIR: systemRoot, PATH: path.join(systemRoot, 'System32'), TEMP: process.env.TEMP, TMP: process.env.TMP },
  }).trim();
  if (!output.startsWith(`speech.cpp ${version}, C API `)) throw new Error(`Unexpected Windows CPU ASR version: ${output}`);
  return { verified, version: output };
}

export function verifyWindowsLLMPackage(directory) {
  directory = path.resolve(directory);
  const binary = path.join(directory, 'llama-server.exe');
  const binaries = [binary, path.join(directory, 'llama-server-avx2.exe')];
  const verified = verifyWindowsPEDependencies(binaries);
  const systemRoot = process.env.SystemRoot;
  if (!systemRoot) throw new Error('Windows GGUF smoke requires SystemRoot');
  const result = spawnSync(binary, ['--version'], {
    cwd: directory, encoding: 'utf8', timeout: 15000, windowsHide: true,
    env: { SystemRoot: systemRoot, WINDIR: systemRoot, PATH: path.join(systemRoot, 'System32'), TEMP: process.env.TEMP, TMP: process.env.TMP },
  });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`Windows GGUF runtime version check failed: ${result.status}`);
  const output = `${result.stdout}${result.stderr}`.trim();
  if (!/^version: 0\.7\.0-dev \(build 11734, commit f3fb2fc/.test(output)) throw new Error(`Unexpected Windows GGUF runtime version: ${output}`);
  return { verified, version: output.split('\n')[0].trim() };
}
