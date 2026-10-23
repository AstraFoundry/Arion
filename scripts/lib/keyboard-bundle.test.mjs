import test from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { appendFileSync, cpSync, mkdtempSync, readFileSync, rmSync } from 'node:fs';
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
    const library = path.join(bundle, 'Contents/Frameworks/librime.1.dylib');
    const originalLibrary = digest(library);
    execFileSync('codesign', ['--force', '--sign', '-', '--identifier', 'si.astrafoundry.arion.keyboard-verification', library], { stdio: 'pipe' });
    execFileSync('codesign', ['--force', '--deep', '--sign', '-', bundle], { stdio: 'pipe' });
    assert.notEqual(digest(library), originalLibrary, 'The signature test must change the dylib bytes.');
    assert.equal(verifyKeyboardBundle({ root, bundle }).signature, 'sealed-verified');
    const dictionary = path.join(bundle, 'Contents/Resources/keyboard/data/pinyin_simp.dict.yaml');
    appendFileSync(dictionary, '\n# Corrupted package fixture.\n');
    assert.throws(() => verifyKeyboardBundle({ root, bundle }), /Keyboard package hash mismatch: .*pinyin_simp\.dict\.yaml/);
    assert.equal(verifyKeyboardBundle({ root, bundle: source }).schema, 'arion_pinyin');
  } finally {
    rmSync(temporary, { recursive: true, force: true });
  }
});
