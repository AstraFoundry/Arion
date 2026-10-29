import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { isDeepStrictEqual } from 'node:util';
import { Driver, requireValue } from './protocol.mjs';
import { digest, hashFile, verifyDriverArtifacts } from './artifacts.mjs';

export function assertObservations(scenario, observations) {
  for (const [key, expected] of Object.entries(scenario.assertions)) {
    const actual = key.split('.').reduce((value, part) => value?.[part], observations);
    requireValue(isDeepStrictEqual(actual, expected), `${scenario.id}: assertion failed at ${key}`);
  }
}

export async function runScenarios(suite, config, directory, output) {
  requireValue(suite.version === 1 && Array.isArray(suite.scenarios) && suite.scenarios.length > 0, 'Invalid scenario suite');
  requireValue(new Set(suite.scenarios.map(s => s.id)).size === suite.scenarios.length, 'Duplicate scenario IDs');
  const artifacts = await verifyDriverArtifacts(config, directory);
  await mkdir(output, { recursive: false, mode: 0o700 });
  const driver = new Driver(config, directory);
  const results = [];
  let provenance = null;
  let failure = null;
  let cleanup = null;
  try {
    const hello = await driver.request({ op: 'hello', purpose: 'e2e', artifacts });
    requireValue(hello.execution === 'live' && isDeepStrictEqual(hello.artifacts, artifacts) && ['darwin', 'win32'].includes(hello.platform) && Array.isArray(hello.capabilities), 'Actual macOS/Windows driver provenance required');
    provenance = hello;
    for (const scenario of suite.scenarios) {
      requireValue(scenario.capabilities.every(capability => hello.capabilities.includes(capability)), `${scenario.id}: required native capability not implemented`);
      // Assertions remain in the harness; driver receives only controlled actions.
      const response = await driver.request({ op: 'scenario', id: scenario.id, actions: scenario.actions });
      requireValue(response.execution === 'live' && response.fixtureOwnership === 'isolated' && response.observations && Array.isArray(response.evidence) && response.evidence.length > 0, `${scenario.id}: native observations/evidence missing`);
      requireValue(response.evidence.every(e => typeof e.path === 'string' && /^[a-f0-9]{64}$/.test(e.sha256)), 'Evidence paths and hashes required');
      // Verify owned trace files, not just strings claimed in a response.
      for (const evidence of response.evidence) {
        const evidencePath = path.resolve(directory, evidence.path);
        requireValue(await hashFile(evidencePath) === evidence.sha256, 'E2E evidence checksum mismatch');
      }
      assertObservations(scenario, response.observations);
      results.push({ id: scenario.id, acceptance: scenario.acceptance, status: 'passed', response });
    }
  } catch (error) {
    failure = error;
  } finally {
    try {
      cleanup = await driver.request({ op: 'cleanup' });
      requireValue(cleanup.execution === 'live' && cleanup.cleanupComplete === true, 'Owned fixture cleanup not confirmed');
    } catch (error) { failure ??= error; }
    await driver.close();
    await writeFile(path.join(output, 'results.json'), JSON.stringify({ version: 1, status: failure ? 'failed' : 'passed', suiteHash: digest(suite), provenance, results, cleanup, reason: failure?.message ?? null }, null, 2), { mode: 0o600 });
  }
  if (failure) throw failure;
  return results.length;
}
