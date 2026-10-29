import assert from 'node:assert/strict';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { windowsPEImports, verifyWindowsPEDependencies } from './windows-pe.mjs';

function peImage(imports = [], delayImports = []) {
  const bytes = Buffer.alloc(0x2200);
  bytes.write('MZ');
  bytes.writeUInt32LE(0x80, 0x3c);
  bytes.write('PE\0\0', 0x80);
  bytes.writeUInt16LE(0x8664, 0x84);
  bytes.writeUInt16LE(1, 0x86);
  bytes.writeUInt16LE(240, 0x94);
  const optional = 0x98;
  bytes.writeUInt16LE(0x20b, optional);
  bytes.writeUInt32LE(0x200, optional + 60);
  bytes.writeUInt32LE(16, optional + 108);
  const section = optional + 240;
  bytes.writeUInt32LE(0x1000, section + 12);
  bytes.writeUInt32LE(0x2000, section + 16);
  bytes.writeUInt32LE(0x200, section + 20);
  let nameOffset = 0x800;
  for (const [names, index, offset, width, field] of [[imports, 1, 0x200, 20, 12], [delayImports, 13, 0x400, 32, 4]]) {
    if (!names.length) continue;
    bytes.writeUInt32LE(offset + 0xe00, optional + 112 + index * 8);
    bytes.writeUInt32LE((names.length + 1) * width, optional + 116 + index * 8);
    names.forEach((name, position) => {
      const descriptor = offset + position * width;
      if (index === 13) bytes.writeUInt32LE(1, descriptor);
      bytes.writeUInt32LE(nameOffset + 0xe00, descriptor + field);
      bytes.write(name, nameOffset, 'ascii');
      nameOffset += name.length + 1;
    });
  }
  return bytes;
}

test('Windows PE reader includes normal and delay-loaded dependencies', () => {
  assert.deepEqual(windowsPEImports(peImage(['KERNEL32.dll', 'VCOMP140.DLL'], ['MSVCP140.dll'])), ['kernel32.dll', 'vcomp140.dll', 'msvcp140.dll']);
});

test('Windows PE reader refuses malformed architectures, ranges and DLL paths', () => {
  const wrongArchitecture = peImage();
  wrongArchitecture.writeUInt16LE(0x14c, 0x84);
  assert.throws(() => windowsPEImports(wrongArchitecture), /x64/);
  assert.throws(() => windowsPEImports(Buffer.alloc(10)), /range/);
  assert.throws(() => windowsPEImports(peImage(['..\\vcomp140.dll'])), /Unsafe/);
  const invalidAddress = peImage(['vcomp140.dll']);
  invalidAddress.writeUInt32LE(0xffffffff, 0x200 + 12);
  assert.throws(() => windowsPEImports(invalidAddress), /outside/);
});

test('Windows dependency gate rejects unbundled compiler runtimes including delay imports', t => {
  const directory = mkdtempSync(path.join(tmpdir(), 'arion-pe-'));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const executable = path.join(directory, 'speech.exe');
  for (const [imports, delay] of [[['VCOMP140.DLL'], []], [[], ['libgomp-1.dll']]]) {
    writeFileSync(executable, peImage(imports, delay));
    assert.throws(() => verifyWindowsPEDependencies([executable]), /Unbundled Windows DLL dependency/);
  }
});

test('Windows dependency gate follows bundled DLLs and limits optional driver imports', t => {
  const directory = mkdtempSync(path.join(tmpdir(), 'arion-pe-'));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const executable = path.join(directory, 'speech-vulkan.exe');
  writeFileSync(executable, peImage(['KERNEL32.dll', 'helper.dll', 'vulkan-1.dll']));
  writeFileSync(path.join(directory, 'HELPER.dll'), peImage(['api-ms-win-core-synch-l1-2-0.dll']));
  assert.throws(() => verifyWindowsPEDependencies([executable]), /vulkan-1.dll/);
  const options = { optionalImports: { 'speech-vulkan.exe': ['vulkan-1.dll'] } };
  assert.equal(verifyWindowsPEDependencies([executable], options).length, 2);
  writeFileSync(path.join(directory, 'HELPER.dll'), peImage(['MSVCP140.dll']));
  assert.throws(() => verifyWindowsPEDependencies([executable], options), /msvcp140.dll/);
});
