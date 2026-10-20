/** Collect rendered heading slugs and explicit HTML anchors from Markdown. */
export function markdownAnchors(source) {
  const text = source.replace(/^\s*(`{3,}|~{3,})[^\n]*\n[\s\S]*?^\s*\1\s*$/gm, '');
  const anchors = new Set();
  const counts = new Map();
  for (const match of text.matchAll(/^#{1,6}\s+(.+)$/gm)) {
    const base = match[1].toLowerCase().replace(/[`*_]/g, '').replace(/[^\p{L}\p{N}\s_-]/gu, '').replace(/\s/g, '-');
    const count = counts.get(base) ?? 0;
    anchors.add(count ? `${base}-${count}` : base);
    counts.set(base, count + 1);
  }
  for (const tag of text.matchAll(/<([a-z][\w:-]*)(\s[^<>]*?)?\s*\/?>/gi)) {
    for (const attribute of (tag[2] ?? '').matchAll(/\s(id|name)\s*=\s*(["'])(.*?)\2/gi)) {
      if (attribute[1].toLowerCase() === 'id' || tag[1].toLowerCase() === 'a') anchors.add(attribute[3]);
    }
  }
  return anchors;
}
