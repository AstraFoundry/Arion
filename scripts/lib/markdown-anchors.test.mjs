import assert from 'node:assert/strict';
import test from 'node:test';
import { markdownAnchors } from './markdown-anchors.mjs';

test('official card HTML anchors resolve alongside generated heading slugs', () => {
  const anchors = markdownAnchors('# Overview\n# Overview\n<a id="quickstart"></a>\n<a name=\'benchmarks\'></a>\n<div class="section" id="highlights">Text</div>');
  assert.deepEqual([...anchors], ['overview', 'overview-1', 'quickstart', 'benchmarks', 'highlights']);
  assert.equal(anchors.has('missing'), false);
});

test('code examples and unrelated attributes do not satisfy a missing link', () => {
  const anchors = markdownAnchors('```html\n<a id="example"></a>\n# Example\n```\n<div data-id="fake" name="not-an-anchor"></div>');
  assert.deepEqual([...anchors], []);
});
