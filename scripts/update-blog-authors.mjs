// Refreshes the blog author cache in apps/site/content/authors from GitHub.
// The site build reads that cache and makes no network request for it.
import { createHash } from 'node:crypto';
import { mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const avatarSize = 64;
const timeoutMs = 20_000;
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const content = path.join(root, 'apps/site/content/blog');
const output = path.join(root, 'apps/site/content/authors');
const LOGIN = /^[a-z\d](?:[a-z\d]|-(?=[a-z\d])){0,38}$/i;

const headers = {
  Accept: 'application/vnd.github+json',
  'User-Agent': 'arion-update-blog-authors',
  'X-GitHub-Api-Version': '2022-11-28',
};
if (process.env.GITHUB_TOKEN?.trim()) headers.Authorization = `Bearer ${process.env.GITHUB_TOKEN.trim()}`;

function unquote(value) {
  const trimmed = value.trim();
  if (
    (trimmed.startsWith('"') && trimmed.endsWith('"')) ||
    (trimmed.startsWith("'") && trimmed.endsWith("'"))
  ) {
    return trimmed.slice(1, -1).trim();
  }
  return trimmed;
}

function splitLogins(value) {
  return unquote(value)
    .replace(/^\[|\]$/g, '')
    .split(',')
    .map((item) => unquote(item))
    .filter(Boolean);
}

function loginsInFile(text, file) {
  const logins = [];
  const fence = text.match(/^---\r?\n([\s\S]*?)\r?\n---/);
  const lines = (fence ? fence[1] : '').split(/\r?\n/);
  for (let i = 0; i < lines.length; i++) {
    const match = lines[i].match(/^authors:\s*(.*)$/);
    if (!match) continue;
    const rest = match[1].trim();
    if (rest === '' || rest === '|' || rest === '>') {
      for (let j = i + 1; j < lines.length; j++) {
        const item = lines[j].match(/^\s+-\s+(.+)$/);
        if (!item) break;
        logins.push(unquote(item[1]));
      }
    } else {
      logins.push(...splitLogins(rest));
    }
  }
  const body = fence ? text.slice(fence[0].length) : text;
  for (const match of body.matchAll(/<github-people\b[^>]*\blogins\s*=\s*(?:"([^"]*)"|'([^']*)')/g)) {
    logins.push(...splitLogins(match[1] ?? match[2]));
  }
  for (const match of body.matchAll(/<voice-quote\b[^>]*\blogin\s*=\s*(?:"([^"]*)"|'([^']*)')/g)) {
    logins.push(unquote(match[1] ?? match[2]));
  }
  for (const login of logins) {
    if (!LOGIN.test(login)) throw new Error(`${file}: "${login}" is not a GitHub login`);
  }
  return logins;
}

function markdownFiles(dir) {
  const files = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const name = path.join(dir, entry.name);
    if (entry.isDirectory()) files.push(...markdownFiles(name));
    else if (entry.name.endsWith('.md')) files.push(name);
  }
  return files;
}

async function fetchOK(url, options) {
  let response;
  try {
    response = await fetch(url, { ...options, signal: AbortSignal.timeout(timeoutMs) });
  } catch (error) {
    throw new Error(`Fetching ${url} failed: ${error.message}`);
  }
  if (response.ok) return response;
  const limited = response.status === 403 || response.status === 429;
  throw new Error(
    `Fetching ${url} failed: HTTP ${response.status}` +
      (limited ? '; set GITHUB_TOKEN to raise the GitHub API rate limit' : ''),
  );
}

const seen = new Map();
for (const file of markdownFiles(content).sort()) {
  for (const login of loginsInFile(readFileSync(file, 'utf8'), path.relative(root, file))) {
    const key = login.toLowerCase();
    if (!seen.has(key)) seen.set(key, login);
  }
}
if (seen.size === 0) throw new Error('No blog post names a GitHub login');

const extensions = { 'image/png': 'png', 'image/jpeg': 'jpg', 'image/gif': 'gif', 'image/webp': 'webp' };
const logins = [...seen.values()].sort((a, b) => {
  const left = a.toLowerCase();
  const right = b.toLowerCase();
  if (left < right) return -1;
  if (left > right) return 1;
  return 0;
});

const downloaded = [];
for (const login of logins) {
  const response = await fetchOK(`https://api.github.com/users/${encodeURIComponent(login)}`, { headers });
  const user = await response.json();
  if (!user.login || !user.html_url || !user.avatar_url) {
    throw new Error(`${login}: GitHub returned a profile without a login, profile URL or avatar`);
  }
  if (user.login.toLowerCase() !== login.toLowerCase()) {
    throw new Error(`${login} is now ${user.login}; update the post to the current GitHub login`);
  }
  const avatarURL = new URL(user.avatar_url);
  avatarURL.searchParams.set('s', String(avatarSize));
  const image = await fetchOK(avatarURL, { headers: { 'User-Agent': headers['User-Agent'] } });
  const type = image.headers.get('content-type')?.split(';')[0]?.trim();
  const extension = extensions[type];
  if (!extension) throw new Error(`${user.login}: unsupported avatar type ${type}`);
  const bytes = Buffer.from(await image.arrayBuffer());
  downloaded.push({
    login: user.login,
    name: user.name?.trim() || user.login,
    profileUrl: user.html_url,
    avatar: `${user.login}.${extension}`,
    avatarSHA256: createHash('sha256').update(bytes).digest('hex'),
    bytes,
  });
}

mkdirSync(output, { recursive: true });
for (const name of readdirSync(output)) {
  if (name !== 'authors.json') rmSync(path.join(output, name));
}
for (const entry of downloaded) writeFileSync(path.join(output, entry.avatar), entry.bytes);
writeFileSync(
  path.join(output, 'authors.json'),
  `${JSON.stringify(
    {
      source: 'https://api.github.com/users/{login}',
      avatarPx: avatarSize,
      authors: downloaded.map(({ bytes, ...entry }) => entry),
    },
    null,
    2,
  )}\n`,
);
console.log(`Wrote ${downloaded.length} blog authors to ${path.relative(root, output)}`);
