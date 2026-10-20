import { execFileSync } from 'node:child_process';
import { existsSync, lstatSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { homedir, tmpdir } from 'node:os';
import path from 'node:path';
import { macosSigningSettings, signingRunner, stripAppleCredentials } from './desktop-signing.mjs';
import { stageIMEProvisioning, verifyIMEProvisioning } from './ime-provisioning.mjs';

export function prepareSystemReset(root, scopes) {
  const idle = { check() {}, execute() {}, cleanup() {} };
  if (!Object.values(scopes).some(Boolean)) return idle;
  if (process.platform !== 'darwin') throw new Error('System reset is implemented only on macOS. Use keep-system on Windows.');
  const directory = mkdtempSync(path.join(tmpdir(), 'arion-reset-'));
  const cleanup = () => rmSync(directory, { recursive: true, force: true });
  const run = (command, args) => execFileSync(command, args, { encoding: 'utf8', env: stripAppleCredentials(process.env) });
  try {
    const identifier = 'si.astrafoundry.inputmethod.arion';
    const installed = path.join(homedir(), 'Library/Input Methods/ArionInput.app');
    if (scopes.inputMethod) {
      for (let current = installed; current !== path.dirname(current); current = path.dirname(current)) {
        if (existsSync(current) && lstatSync(current).isSymbolicLink()) throw new Error(`Refusing a symlinked input-method path: ${current}`);
      }
    }
    if (scopes.inputMethod && existsSync(installed)) {
      const actual = run('/usr/bin/plutil', ['-extract', 'CFBundleIdentifier', 'raw', '-o', '-', path.join(installed, 'Contents/Info.plist')]).trim();
      if (actual !== identifier) throw new Error('Installed input method does not belong to Arion.');
    }
    const bundle = path.join(directory, 'ArionReset.app');
    const executable = path.join(bundle, 'Contents/MacOS/ArionReset');
    mkdirSync(path.dirname(executable), { recursive: true });
    const config = JSON.parse(readFileSync(path.join(root, 'apps/desktop/desktop.json'), 'utf8'));
    const plist = path.join(bundle, 'Contents/Info.plist');
    writeFileSync(plist, JSON.stringify({ CFBundleIdentifier: identifier, CFBundleExecutable: 'ArionReset', CFBundleName: 'Arion Reset', CFBundlePackageType: 'APPL', CFBundleInfoDictionaryVersion: '6.0', LSUIElement: true, LSMinimumSystemVersion: config.macos.minimumSystemVersion }));
    run('/usr/bin/plutil', ['-convert', 'xml1', plist]);
    const native = path.join(root, 'apps/desktop/infrastructure/macos');
    run('/usr/bin/xcrun', ['clang', '-fobjc-arc', '-fblocks', '-I', native,
      path.join(root, 'scripts/native/reset-desktop.m'), path.join(native, 'native.m'),
      '-framework', 'Cocoa', '-framework', 'Carbon', '-framework', 'Security', '-framework', 'LocalAuthentication', '-framework', 'AVFoundation', '-framework', 'ApplicationServices', '-o', executable]);
    if (scopes.keyboard) {
      const settings = macosSigningSettings(process.env);
      if (!settings?.imeProvisioningProfile) throw new Error('Deleting the keyboard key requires MACOS_SIGNING_IDENTITY and MACOS_IME_PROVISIONING_PROFILE with an installed Developer ID identity. Use keep-keyboard or keep-system to retain it.');
      const entitlements = path.join(directory, 'reset.entitlements.plist');
      stageIMEProvisioning({ profilePath: settings.imeProvisioningProfile, identity: settings.identity, bundleIdentifier: identifier, bundle, entitlementsFile: entitlements });
      const sign = signingRunner([settings.certificate, settings.certificatePassword, settings.appPassword]);
      sign('/usr/bin/codesign', ['--force', '--sign', settings.identity, '--options', 'runtime', '--timestamp', '--entitlements', entitlements, bundle]);
      sign('/usr/bin/codesign', ['--verify', '--strict', bundle]);
      verifyIMEProvisioning({ bundle, identity: settings.identity });
    } else {
      run('/usr/bin/codesign', ['--force', '--sign', '-', bundle]);
    }
    const args = Object.entries(scopes).filter(([, enabled]) => enabled).map(([scope]) => scope);
    return {
      check() { run(executable, ['check', ...args]); },
      execute() {
        run(executable, ['execute', ...args]);
        if (scopes.inputMethod) rmSync(installed, { recursive: true, force: true });
        if (scopes.permissions) {
          for (const id of ['si.astrafoundry.arion', identifier]) run('/usr/bin/tccutil', ['reset', 'All', id]);
        }
      },
      cleanup,
    };
  } catch (error) {
    cleanup();
    throw error;
  }
}
