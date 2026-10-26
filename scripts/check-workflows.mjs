import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { checkWorkflows } from './lib/workflow-check.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
process.exitCode = checkWorkflows(root);
