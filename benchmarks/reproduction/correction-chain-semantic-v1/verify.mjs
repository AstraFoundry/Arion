import { verify } from '../correction-chain-v1/archive.mjs';
import { fileURLToPath } from 'node:url';
console.log(JSON.stringify(verify(fileURLToPath(new URL('../../runs/correction-chain-semantic-v1/manifest.json', import.meta.url)))));
