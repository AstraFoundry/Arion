import { readFile } from 'node:fs/promises';
const registry = JSON.parse(await readFile(new URL('../benchmarks/datasets.json', import.meta.url), 'utf8'));
const results = await Promise.allSettled(registry.datasets.map(async dataset => {
  if (!/^[\w.-]+\/[\w.-]+$/.test(dataset.id) || !/^[a-f0-9]{40}$/.test(dataset.revision)) throw new Error('Invalid pinned dataset identity');
  let response;
  try { response = await fetch(`https://huggingface.co/api/datasets/${dataset.id}/revision/${dataset.revision}`, { signal: AbortSignal.timeout(30000) }); } catch (error) { throw new Error(`${dataset.id}: metadata fetch failed (${error.cause?.code ?? error.name})`); }
  if (!response.ok) throw new Error(`${dataset.id}: metadata HTTP ${response.status}`);
  const data = await response.json();
  if (data.sha !== dataset.revision) throw new Error(`${dataset.id}: pinned revision mismatch`);
  return `${dataset.id}: pinned metadata exists; access=${String(data.gated)}; review=${dataset.licenseStatus}`;
}));
for (const result of results) {
  if (result.status === 'fulfilled') console.log(result.value);
  else { console.error(result.reason.message); process.exitCode = 1; }
}
console.log('Metadata check only: no audio downloaded, no access terms accepted, no model evaluated.');
