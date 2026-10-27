// Validate the pinned Emoji index without a network or Python dependency.
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const source = path.join(root, 'apps/desktop/ime/shared');
const data = path.join(root, 'scripts/data/emoji');
const manifest = JSON.parse(readFileSync(path.join(source, 'emoji_sources.json'), 'utf8'));
const hash = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const requireDigest = (file, expected) => {
  if (hash(file) !== expected) throw new Error(`Pinned Emoji digest mismatch: ${file}`);
};

if (manifest.cldrVersion !== '48.2' || manifest.emojiVersion !== '15.0' || manifest.license !== 'Unicode-3.0') {
  throw new Error('Unexpected Emoji data provenance');
}
for (const [name, spec] of Object.entries(manifest.sources)) requireDigest(path.join(data, name), spec.sha256);
const aliasPath = path.join(source, manifest.aliases.file);
const generatedPath = path.join(source, manifest.generated.file);
requireDigest(aliasPath, manifest.aliases.sha256);
requireDigest(generatedPath, manifest.generated.sha256);

const qualified = new Set();
for (const line of readFileSync(path.join(data, 'emoji-test-15.0.txt'), 'utf8').split('\n')) {
  const match = line.match(/^([0-9A-F ]+)\s*; fully-qualified\s+#/);
  if (match) qualified.add(match[1].trim().split(/\s+/).map(code => String.fromCodePoint(parseInt(code, 16))).join(''));
}
const reviewed = new Map(JSON.parse(readFileSync(aliasPath, 'utf8')).map(item => [item.trigger, item.emoji]));
if (reviewed.size !== manifest.aliases.count) throw new Error('Reviewed Emoji alias count changed');

const mappings = new Map();
const metadataHeader = readFileSync(path.join(source, 'emoji.h'), 'utf8');
const capacity = name => {
  const match = metadataHeader.match(new RegExp(`\\b${name} = ([0-9]+)`));
  if (!match) throw new Error(`Missing Emoji metadata capacity: ${name}`);
  return Number(match[1]);
};
const sourceCapacity = capacity('ARION_EMOJI_SOURCE_CAPACITY');
const sequenceCapacity = capacity('ARION_EMOJI_SEQUENCE_ID_CAPACITY');
let previous = null;
for (const line of readFileSync(generatedPath, 'utf8').split('\n')) {
  if (!line.startsWith('    {')) continue;
  if (!line.endsWith('},')) throw new Error('Malformed generated Emoji row');
  const [trigger, emoji, id] = JSON.parse(`[${line.slice(5, -2)}]`);
  if (Buffer.byteLength(trigger) >= sourceCapacity || Buffer.byteLength(id) >= sequenceCapacity) {
    throw new Error(`Emoji metadata exceeds the native ABI capacity: ${trigger}`);
  }
  if (previous !== null && Buffer.compare(Buffer.from(previous), Buffer.from(trigger)) >= 0) {
    throw new Error('Generated Emoji triggers are not unique and sorted');
  }
  if (!reviewed.has(trigger) && !/^[\u3400-\u9fff]{2,6}$/u.test(trigger)) {
    throw new Error(`Invalid automatic Emoji trigger: ${trigger}`);
  }
  if (!qualified.has(emoji) || [...emoji].some(char => {
    const code = char.codePointAt(0);
    return code >= 0x1f3fb && code <= 0x1f3ff;
  })) throw new Error(`Invalid generated Emoji sequence: ${trigger}`);
  if (id !== [...emoji].map(char => char.codePointAt(0).toString(16).toUpperCase()).join('-')) {
    throw new Error(`Invalid generated Emoji sequence ID: ${trigger}`);
  }
  mappings.set(trigger, emoji);
  previous = trigger;
}
if (mappings.size !== manifest.generated.mappings) throw new Error('Generated Emoji mapping count differs');
for (const [trigger, emoji] of reviewed) {
  if (mappings.get(trigger) !== emoji) throw new Error(`Reviewed Emoji override changed: ${trigger}`);
}
if (mappings.get('中国') !== '🇨🇳' || mappings.get('手机') !== '📱' || ['天气', '工作', '服务', '注册', '周末', '一点', '笔记本'].some(word => mappings.has(word))) {
  throw new Error('Emoji ambiguity regression');
}
console.log(`Verified ${mappings.size} pinned Emoji mappings and ${reviewed.size} reviewed overrides.`);
