import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { readdir, readFile, stat } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { decisionErrors, makeTargets, missingTranslatedLinks, unknownCommands } from './lib/doc-consistency.mjs';
import { markdownAnchors } from './lib/markdown-anchors.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const errors = [];
const files = [];
const trees = ['docs', 'specs', 'apps', 'packages', 'services', 'models', 'deploy', 'tools', 'tests', 'scripts'];
const ignored = new Set(['build', 'dist', 'node_modules', '.git', '.next', 'out', '.mygo']);
const generated = new Set(['apps/desktop/resources']);

async function walk(relative) {
  for (const entry of await readdir(path.join(root, relative), { withFileTypes: true })) {
    if (ignored.has(entry.name) || entry.isSymbolicLink()) continue;
    const name = path.join(relative, entry.name);
    if (entry.isDirectory() && !generated.has(name.split(path.sep).join('/'))) await walk(name);
    else files.push(name);
  }
}
await Promise.all(trees.map(walk));
files.push('README.md', 'README.zh-CN.md', 'README.ja.md', 'CLAUDE.md', 'package.json');

// Blog posts link with site URLs (`/privacy/`, `/blog/<slug>/chart.webp`):
// each must be a static file in the site's public/ or an English route.
const siteContent = path.join('apps', 'site', 'content') + path.sep;
async function siteTargetExists(url) {
  const site = path.join(root, 'apps', 'site');
  const decoded = decodeURIComponent(url);
  const candidates = [
    path.join(site, 'public', decoded),
    path.join(site, 'app', '(en)', decoded, 'page.tsx'),
  ];
  const post = decoded.match(/^\/blog\/([^/]+)\/$/);
  if (post) candidates.push(path.join(site, 'content', 'blog', post[1], 'en.md'));
  for (const candidate of candidates) {
    try {
      if ((await stat(candidate)).isFile()) return true;
    } catch (error) {
      // Only an absent candidate means "try the next one"; other errors are real.
      if (error.code !== 'ENOENT' && error.code !== 'ENOTDIR') throw error;
    }
  }
  return false;
}

const markdown = new Map();
const rawMarkdown = new Map();
const linked = new Set();
for (const name of files.filter(name => name.endsWith('.md'))) {
  const source = await readFile(path.join(root, name), 'utf8');
  rawMarkdown.set(name, source);
  markdown.set(name, source.replace(/^\s*(`{3,}|~{3,})[^\n]*\n[\s\S]*?^\s*\1\s*$/gm, ''));
}


for (const [name, source] of markdown) {
  for (const match of source.matchAll(/!?\[[^\]\n]*\]\((<[^>]+>|[^\s)]+)(?:\s+"[^"]*")?\)/g)) {
    const target = match[1].replace(/^<|>$/g, '');
    if (/^[a-z][a-z\d+.-]*:/i.test(target) || target.startsWith('//')) continue;
    const [file, fragment] = target.split('#', 2);
    if (name.startsWith(siteContent) && file.startsWith('/')) {
      if (!(await siteTargetExists(file))) errors.push(`${name}: missing site page or file ${target}`);
      continue;
    }
    const resolved = file ? path.resolve(root, path.dirname(name), decodeURIComponent(file)) : path.join(root, name);
    try {
      const info = await stat(resolved);
      if (info.isFile() && resolved !== path.join(root, name)) linked.add(path.relative(root, resolved).split(path.sep).join('/'));
      if (fragment && info.isFile() && resolved.endsWith('.md')) {
        const text = await readFile(resolved, 'utf8');
        if (!markdownAnchors(text).has(decodeURIComponent(fragment))) errors.push(`${name}: missing anchor ${target}`);
      }
    } catch {
      errors.push(`${name}: missing local target ${target}`);
    }
  }
  if (name !== path.join('docs', 'decisions', 'adr-template.md') && /\b__[A-Z][A-Z_]+__\b/.test(source)) {
    errors.push(`${name}: unresolved template placeholder`);
  }
}
for (const name of files.filter(name => name.endsWith('.json'))) {
  try { JSON.parse(await readFile(path.join(root, name), 'utf8')); }
  catch (error) { errors.push(`${name}: invalid JSON (${error.message})`); }
}

const rootPackage = JSON.parse(await readFile(path.join(root, 'package.json'), 'utf8'));
if (!/^pnpm@\d+\.\d+\.\d+$/.test(rootPackage.packageManager)) errors.push('Root packageManager must pin an exact pnpm version');
if (rootPackage.scripts && Object.values(rootPackage.scripts).some(command => /\b(npm|npx)\s/.test(command))) errors.push('Project scripts must use pnpm');
for (const command of Object.values(rootPackage.scripts ?? {})) {
  if (command.includes('benchmarks/') || command.includes('tools/evaluation')) errors.push('Public commands cannot depend on private research paths');
}

// Documented commands must exist. Frozen research evidence under benchmarks/
// and hash-pinned upstream notices retain their original command context.
const targets = makeTargets(await readFile(path.join(root, 'Makefile'), 'utf8'));
const keyboardManifest = JSON.parse(await readFile(path.join(root, 'scripts/keyboard-runtime.json'), 'utf8'));
const upstreamDocuments = new Map(keyboardManifest.files
  .filter(file => typeof file.path === 'string' && file.path.endsWith('.md') && file.source && file.sha256)
  .map(file => [file.path, file.sha256]));
for (const [name, source] of rawMarkdown) {
  if (name.startsWith('benchmarks' + path.sep)) continue;
  const upstreamHash = upstreamDocuments.get(name.split(path.sep).join('/'));
  if (upstreamHash) {
    const actual = createHash('sha256').update(await readFile(path.join(root, name))).digest('hex');
    if (actual !== upstreamHash) errors.push(`${name}: upstream documentation differs from its pinned bytes`);
    continue;
  }
  errors.push(...unknownCommands(name, source, rootPackage.scripts, targets));
}

// Every document under docs/ is reachable from another document, and ADR
// numbers are unique and indexed.
for (const name of files.filter(name => name.endsWith('.md') && name.startsWith('docs' + path.sep))) {
  const normalized = name.split(path.sep).join('/');
  if (!linked.has(normalized)) errors.push(`${name}: not linked from any other Markdown document`);
}
const decisionNames = (await readdir(path.join(root, 'docs', 'decisions'))).filter(name => /^\d{3}-/.test(name));
errors.push(...decisionErrors(decisionNames, rawMarkdown.get(path.join('docs', 'decisions', 'README.md'))));

// The translated READMEs state the same facts: same local links as README.md.
for (const name of ['README.zh-CN.md', 'README.ja.md']) {
  errors.push(...missingTranslatedLinks(name, rawMarkdown.get(name), 'README.md', rawMarkdown.get('README.md')));
}
// AGENTS.md mirrors CLAUDE.md: a symlink, or on checkouts without symlink support a copy.
const agents = (await readFile(path.join(root, 'AGENTS.md'), 'utf8')).trim();
if (agents !== 'CLAUDE.md' && agents !== (await readFile(path.join(root, 'CLAUDE.md'), 'utf8')).trim()) errors.push('AGENTS.md must be a symlink to CLAUDE.md or an identical copy');

for (const name of files.filter(name => name.endsWith('.mjs'))) {
  const checked = spawnSync(process.execPath, ['--check', path.join(root, name)], { encoding: 'utf8' });
  if (checked.error || checked.status !== 0) errors.push(`${name}: JavaScript syntax check failed`);
}

const layers = ['shared', 'entities', 'features', 'widgets', 'pages', 'app'];
for (const name of files.filter(name => name.endsWith('.go'))) {
  const source = await readFile(path.join(root, name), 'utf8');
  const imports = [];
  for (const match of source.matchAll(/^import\s+(?:\(([\s\S]*?)\)|(?:[\w.]+\s+)?"([^"]+)")/gm)) {
    if (match[2]) imports.push(match[2]);
    else for (const item of match[1].matchAll(/"([^"]+)"/g)) imports.push(item[1]);
  }
  const normalized = name.split(path.sep).join('/');
  const ui = normalized.match(/^apps\/desktop\/ui\/([^/]+)\/([^/]+)/);
  for (const dependency of imports) {
    if (normalized.startsWith('packages/core/')) {
      if (!dependency.startsWith('arion.local/core/') && dependency.split('/')[0].includes('.')) errors.push(`${name}: third-party dependency in portable core: ${dependency}`);
      if (/^(os|syscall|net|database|plugin)(\/|$)/.test(dependency)) errors.push(`${name}: OS/transport dependency in portable core: ${dependency}`);
      if (normalized.startsWith('packages/core/domain/') && /arion\.local\/core\/(application|infrastructure|interfaces)\//.test(dependency)) errors.push(`${name}: outward domain import: ${dependency}`);
      if (normalized.startsWith('packages/core/application/') && /arion\.local\/core\/(infrastructure|interfaces)\//.test(dependency)) errors.push(`${name}: concrete adapter import in application: ${dependency}`);
      if (normalized.startsWith('packages/core/contracts/') && /arion\.local\/core\/(application|infrastructure|interfaces)\//.test(dependency)) errors.push(`${name}: outward portable contract import: ${dependency}`);
    }
    if (normalized.startsWith('services/sync/domain/')) {
      if (dependency.includes('.') || /^(net|database|os|syscall|plugin)(\/|$)/.test(dependency)) errors.push(`${name}: non-stdlib or transport dependency in sync domain: ${dependency}`);
      if (/arion\.local\/sync\/(application|infrastructure|interfaces)\//.test(dependency)) errors.push(`${name}: outward sync domain import: ${dependency}`);
    }
    if (normalized.startsWith('services/sync/application/') && /arion\.local\/sync\/(infrastructure|interfaces)\//.test(dependency)) errors.push(`${name}: concrete adapter import in sync application: ${dependency}`);
    if (normalized.startsWith('services/updates/domain/')) {
      if (dependency.includes('.') || /^(net|database|os|syscall|plugin)(\/|$)/.test(dependency)) errors.push(`${name}: non-stdlib or transport dependency in updates domain: ${dependency}`);
      if (/arion\.local\/updates\/(application|infrastructure|interfaces)\//.test(dependency)) errors.push(`${name}: outward updates domain import: ${dependency}`);
    }
    if (normalized.startsWith('services/updates/application/') && /arion\.local\/updates\/(infrastructure|interfaces)\//.test(dependency)) errors.push(`${name}: concrete adapter import in updates application: ${dependency}`);
    if (!ui) continue;
    if (dependency.startsWith('arion.local/desktop/') && !dependency.startsWith('arion.local/desktop/ui/')) errors.push(`${name}: concrete desktop dependency in UI: ${dependency}`);
    const target = dependency.match(/^arion\.local\/desktop\/ui\/([^/]+)\/([^/]+)/);
    if (target) {
      const current = layers.indexOf(ui[1]);
      const imported = layers.indexOf(target[1]);
      // `shared` has segments, not slices: its segments may import each other.
      if (current < 0 || imported < 0 || imported > current || (imported === current && ui[2] !== target[2] && ui[1] !== 'shared')) errors.push(`${name}: upward/cross-slice UI import: ${dependency}`);
    }
    if (dependency.startsWith('arion.local/core/') && ui[1] !== 'shared') errors.push(`${name}: core values must enter UI through shared boundary: ${dependency}`);
  }
}
if (errors.length) {
  for (const error of errors) console.error(error);
  process.exitCode = 1;
} else {
  console.log(`Repository checks passed: ${markdown.size} Markdown documents, JSON, JavaScript syntax and Go boundaries.`);
}
