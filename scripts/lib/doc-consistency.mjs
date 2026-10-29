// Mechanical documentation checks: a documented command must exist, ADR
// numbers are unique and indexed, and the translated READMEs link what the
// English one links.
const conventionalSubject = /^[a-z]+(?:\([a-z0-9-]+\))?!?: /;
const pnpmBuiltins = new Set(['install', 'exec', 'dlx', 'add', 'run', 'approve-builds', 'update', 'list', 'config', 'store', 'why', 'outdated', 'audit', 'rebuild', 'prune', 'remove']);

/** Makefile goals that carry a `##` help comment or are declared as rules. */
export function makeTargets(makefile) {
  return new Set([...makefile.matchAll(/^([A-Za-z0-9][\w.-]*):/gm)].map(match => match[1]));
}

/** `pnpm <script>` and `make <target>` inside code spans and fenced blocks. */
export function commandReferences(source) {
  const references = [];
  let fence = null;
  source.split('\n').forEach((line, index) => {
    const opening = line.match(/^\s*(`{3,}|~{3,})/);
    if (opening) {
      fence = fence === null ? opening[1] : line.trim().startsWith(fence) ? null : fence;
      return;
    }
    const spans = fence === null ? [...line.matchAll(/`([^`\n]+)`/g)].map(match => match[1]) : [line];
    for (const raw of spans) {
      // A quoted Conventional Commits subject is not a command.
      if (fence === null && conventionalSubject.test(raw)) continue;
      const span = raw.replace(/\s#.*$/, '');
      for (const match of span.matchAll(/(?:^|[\s;&|(])(pnpm|make)\s+(?:run\s+)?([a-z][a-z0-9:_-]*)(?:\{([a-z0-9,:_-]+)\})?/g)) {
        const [, tool, target, alternatives] = match;
        if (tool === 'pnpm' && (pnpmBuiltins.has(target) || span.includes('--filter'))) continue;
        // `pnpm group:{report,test}` names several scripts at once.
        for (const name of alternatives ? alternatives.split(',').map(alternative => target + alternative) : [target]) {
          references.push({ line: index + 1, tool, target: name });
        }
      }
    }
  });
  return references;
}

export function unknownCommands(name, source, scripts, targets) {
  return commandReferences(source)
    .filter(({ tool, target }) => (tool === 'pnpm' ? !Object.hasOwn(scripts, target) : !targets.has(target)))
    .map(({ line, tool, target }) => `${name}:${line}: documented command does not exist: ${tool} ${target}`);
}

/** ADR file names must carry unique numbers and each must appear in the index. */
export function decisionErrors(fileNames, index) {
  const errors = [];
  const byNumber = new Map();
  for (const file of fileNames) {
    const match = file.match(/^(\d{3})-.+\.md$/);
    if (!match) continue;
    if (!byNumber.has(match[1])) byNumber.set(match[1], []);
    byNumber.get(match[1]).push(file);
    if (!index.includes(`(${file})`)) errors.push(`docs/decisions/README.md: ${file} is not listed`);
  }
  for (const [number, files] of byNumber) {
    if (files.length > 1) errors.push(`docs/decisions: ADR number ${number} is used by ${files.join(', ')}`);
  }
  return errors;
}

/** Local link targets of a Markdown document, without anchors. */
export function localLinkTargets(source) {
  const targets = new Set();
  for (const match of source.matchAll(/!?\[[^\]\n]*\]\((<[^>]+>|[^\s)]+)(?:\s+"[^"]*")?\)/g)) {
    const target = match[1].replace(/^<|>$/g, '');
    if (/^[a-z][a-z\d+.-]*:/i.test(target) || target.startsWith('//')) continue;
    targets.add(target.split('#', 1)[0] || `#${target.slice(1)}`);
  }
  return targets;
}

export function missingTranslatedLinks(name, source, referenceName, referenceSource) {
  const have = localLinkTargets(source);
  const want = localLinkTargets(referenceSource);
  const errors = [];
  for (const target of want) if (!have.has(target)) errors.push(`${name}: does not link ${target}, which ${referenceName} links`);
  for (const target of have) if (!want.has(target)) errors.push(`${name}: links ${target}, which ${referenceName} does not link`);
  return errors;
}
