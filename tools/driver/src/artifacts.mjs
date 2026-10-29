import { createHash } from 'node:crypto';
import { createReadStream } from 'node:fs';
import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { requireValue } from './protocol.mjs';

export const sha256 = bytes => createHash('sha256').update(bytes).digest('hex');
export const digest = value => sha256(JSON.stringify(value));
export const readJson = async filename => JSON.parse(await readFile(filename, 'utf8'));
export async function hashFile(filename) {
  const hash = createHash('sha256');
  for await (const chunk of createReadStream(filename)) hash.update(chunk);
  return hash.digest('hex');
}

export async function verifyDriverArtifacts(driverConfig, directory) {
  requireValue(Array.isArray(driverConfig.artifacts) && driverConfig.artifacts.length > 0, 'Driver must pin local model/runtime artifacts');
  const ids = new Set();
  const artifacts = {};
  for (const artifact of driverConfig.artifacts) {
    requireValue(typeof artifact.id === 'string' && !ids.has(artifact.id) && typeof artifact.path === 'string' && /^[a-f0-9]{64}$/.test(artifact.sha256) && typeof artifact.license === 'string' && artifact.license.length > 0, 'Invalid artifact identity/hash/license');
    ids.add(artifact.id);
    requireValue(await hashFile(path.resolve(directory, artifact.path)) === artifact.sha256, `Artifact checksum mismatch for ${artifact.id}`);
    artifacts[artifact.id] = artifact.sha256;
  }
  return artifacts;
}
