import test from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { appendFileSync, copyFileSync, cpSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { verifyInputMethodIdentity, verifyKeyboardBundle } from '../verify-keyboard-bundle.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const digest = file => createHash('sha256').update(readFileSync(file)).digest('hex');

test('shared input-source identity binds both adapter aliases and packaged mode', () => {
  const options = {
    bundleId: 'si.astrafoundry.inputmethod.arion', sourceId: 'si.astrafoundry.inputmethod.arion.pinyin',
    adapter: readFileSync(path.join(root, 'apps/desktop/infrastructure/macos/ime.go'), 'utf8'),
    identity: readFileSync(path.join(root, 'apps/desktop/infrastructure/imebridge/identity.go'), 'utf8'),
  };
  assert.doesNotThrow(() => verifyInputMethodIdentity(options));
  for (const changed of [
    { sourceId: options.bundleId + '.wrong' },
    { bundleId: 'si.astrafoundry.inputmethod.wrong' },
    { identity: options.identity.replace('.pinyin', '.wrong') },
    { adapter: options.adapter.replace('const IMESourceID = imebridge.SourceID', 'const IMESourceID = "wrong"') },
    { adapter: options.adapter.replace('const IMEBundleID = imebridge.BundleID', 'const IMEBundleID = "wrong"') },
  ]) assert.throws(() => verifyInputMethodIdentity({ ...options, ...changed }), /Input mode identity must match/);
});

test('IMK package accepts valid signature changes and rejects corrupted dictionary bytes', { skip: process.platform !== 'darwin' }, () => {
  const source = path.join(root, 'apps/desktop/resources/darwin/ArionInput.app');
  assert.equal(verifyKeyboardBundle({ root, bundle: source }).schema, 'arion_pinyin');
  const temporary = mkdtempSync(path.join(os.tmpdir(), 'arion-keyboard-package-'));
  const bundle = path.join(temporary, 'ArionInput.app');
  try {
    cpSync(source, bundle, { recursive: true });
    const metadata=path.join(bundle,'Contents/Resources/keyboard/runtime.json');
    const originalMetadata=readFileSync(metadata);
    for(const mutate of [
      runtime=>{delete runtime.build;},
      runtime=>{runtime.build.correctionIdentity.version=0;},
      runtime=>{runtime.build.sources.source.sha256='0'.repeat(64);},
      runtime=>{runtime.build.builderSha256='0'.repeat(64);},
      runtime=>{runtime.build.minimumSystemVersion='1.0';},
      runtime=>{runtime.build.fingerprint='0'.repeat(64);},
      runtime=>{runtime.build.outputs={};},
    ]) {
      const runtime=JSON.parse(originalMetadata);mutate(runtime);
      writeFileSync(metadata,JSON.stringify(runtime));
      assert.throws(()=>verifyKeyboardBundle({root,bundle}),/Keyboard (correction identity|source build)/);
    }
    writeFileSync(metadata,originalMetadata);
    const library = path.join(bundle, 'Contents/Frameworks/librime.1.dylib');
    const originalLibrary = digest(library);
    execFileSync('codesign', ['--force', '--sign', '-', '--identifier', 'si.astrafoundry.arion.keyboard-verification', library], { stdio: 'pipe' });
    execFileSync('codesign', ['--force', '--deep', '--sign', '-', bundle], { stdio: 'pipe' });
    assert.notEqual(digest(library), originalLibrary, 'The signature test must change the dylib bytes.');
    assert.equal(verifyKeyboardBundle({ root, bundle }).signature, 'sealed-verified');
    for (const name of ['pinyin_simp.dict.yaml', 'arion_english.lexicon', 'build/pinyin_simp.prism.bin']) {
      const relative = path.join('Contents/Resources/keyboard/data', name);
      const dictionary = path.join(bundle, relative);
      appendFileSync(dictionary, '\n# Corrupted package fixture.\n');
      assert.throws(() => verifyKeyboardBundle({ root, bundle }), error =>
        error.message.includes('Keyboard package hash mismatch:') && error.message.includes(name));
      copyFileSync(path.join(source, relative), dictionary);
    }
    assert.equal(verifyKeyboardBundle({ root, bundle: source }).schema, 'arion_pinyin');
  } finally {
    rmSync(temporary, { recursive: true, force: true });
  }
});
