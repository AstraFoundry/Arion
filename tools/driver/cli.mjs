import { mkdir } from 'node:fs/promises';
import path from 'node:path';
import { readJson } from './src/artifacts.mjs';
import { runScenarios } from './src/e2e.mjs';

try {
  if (process.argv[2] !== 'e2e') throw new Error('Use e2e --suite <file> --driver <file> --out <new-directory>');
  const options = {};
  const allowed = new Set(['suite', 'driver', 'out']);
  for (let index = 3; index < process.argv.length; index += 2) {
    const flag = process.argv[index];
    const value = process.argv[index + 1];
    const key = flag?.slice(2);
    if (!flag?.startsWith('--') || !allowed.has(key) || !value || value.startsWith('--') || Object.hasOwn(options, key)) {
      throw new Error('Use unique --suite, --driver and --out value pairs');
    }
    options[key] = path.resolve(value);
  }
  for (const name of allowed) if (!options[name]) throw new Error(`--${name} is required; no native driver is supplied`);
  const suite = await readJson(options.suite);
  const config = await readJson(options.driver);
  await mkdir(path.dirname(options.out), { recursive: true, mode: 0o700 });
  const count = await runScenarios(suite, config, path.dirname(options.driver), options.out);
  console.log(`Passed ${count} actual OS scenarios.`);
} catch (error) {
  console.error(`Product scenarios failed: ${error.message}`);
  process.exitCode = 1;
}
