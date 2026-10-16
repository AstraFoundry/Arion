import { readFileSync } from 'node:fs';
import { macosBuildConfig } from '../../scripts/lib/desktop-signing.mjs';

// Public package metadata stays readable by release and resource tooling.
const config = JSON.parse(readFileSync(new URL('./desktop.json', import.meta.url), 'utf8'));
export default ({ command }) => macosBuildConfig(config, { command });
