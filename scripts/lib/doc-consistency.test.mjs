import assert from 'node:assert/strict';
import test from 'node:test';
import { commandReferences, decisionErrors, makeTargets, missingTranslatedLinks, unknownCommands } from './doc-consistency.mjs';

const scripts = { check: 'x', 'site:check': 'x', 'rewrite:demo:report': 'x', 'rewrite:demo:test': 'x' };
const targets = makeTargets('.PHONY: build\nbuild: ## Build\n\tnode x\nreset: ## Reset\n\tnode y\n');

test('code spans and fenced blocks name commands; prose does not', () => {
  const source = 'Use the pinned pnpm workspace and make sure.\nRun `pnpm check` then `make build desktop`.\n```sh\nmake reset keep-models # clear data\npnpm --filter @arion/site dev\n```\n';
  assert.deepEqual(commandReferences(source), [
    { line: 2, tool: 'pnpm', target: 'check' },
    { line: 2, tool: 'make', target: 'build' },
    { line: 4, tool: 'make', target: 'reset' },
  ]);
});

test('brace groups expand into one script per alternative', () => {
  assert.deepEqual(commandReferences('`pnpm rewrite:demo:{report,test}`').map(reference => reference.target), ['rewrite:demo:report', 'rewrite:demo:test']);
});

test('a removed pnpm script or make target is reported with its line', () => {
  const source = 'Fine: `pnpm site:check`.\nRemoved: `make desktop-build` and `pnpm site:build`.\n';
  assert.deepEqual(unknownCommands('doc.md', source, scripts, targets), [
    'doc.md:2: documented command does not exist: make desktop-build',
    'doc.md:2: documented command does not exist: pnpm site:build',
  ]);
});

test('pnpm built-ins and workspace filters are not scripts', () => {
  assert.deepEqual(unknownCommands('doc.md', '`pnpm install --frozen-lockfile` `pnpm exec node x.mjs` `pnpm --filter @arion/site dev`', scripts, targets), []);
});

test('ADR numbers are unique and every ADR is indexed', () => {
  const index = '[1](001-a.md) [2](002-b.md)';
  assert.deepEqual(decisionErrors(['001-a.md', '002-b.md', 'README.md', 'adr-template.md'], index), []);
  assert.deepEqual(decisionErrors(['001-a.md', '002-b.md', '002-c.md'], index), [
    'docs/decisions/README.md: 002-c.md is not listed',
    'docs/decisions: ADR number 002 is used by 002-b.md, 002-c.md',
  ]);
});

test('translated READMEs must link what the English README links', () => {
  const english = '[a](docs/a.md#x) [b](docs/b.md) [ext](https://example.com)';
  assert.deepEqual(missingTranslatedLinks('zh.md', '[a](docs/a.md) [b](docs/b.md)', 'en.md', english), []);
  assert.deepEqual(missingTranslatedLinks('zh.md', '[a](docs/a.md) [c](docs/c.md)', 'en.md', english), [
    'zh.md: does not link docs/b.md, which en.md links',
    'zh.md: links docs/c.md, which en.md does not link',
  ]);
});
