import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { validateExperiment } from '../src/manifest.mjs';
import { inferenceRequest, plan } from '../src/runner.mjs';

const config = JSON.parse(await readFile(new URL('../../../benchmarks/configs/desktop-models.json', import.meta.url), 'utf8'));

test('model comparison varies only the pinned bundle', () => {
  validateExperiment(config);
  const features = new Set(config.variants.map(v => JSON.stringify(v.features)));
  assert.equal(features.size, 1, 'treatments must be identical across bundles');
  const sample = { id: 'a', audio: { path: 'a.wav', sha256: 'a'.repeat(64), durationMs: 1 }, context: '', approvedMemory: [] };
  const block = plan(config).blocks.find(b => b.variant.id !== config.baseline);
  assert.equal(inferenceRequest(sample, block, config, '/tmp/c.json').pipeline.asr, block.variant.pipelineOverrides.asr);
});

test('malformed bundle overrides are rejected', () => {
  for (const asr of ['', 'bad id', 7]) {
    const broken = structuredClone(config);
    broken.variants[1].pipelineOverrides = { asr };
    assert.throws(() => validateExperiment(broken), /override/);
  }
});
