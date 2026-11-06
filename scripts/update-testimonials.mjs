// Refreshes the testimonial profile cache in apps/site/content/testimonials
// from X, through FxTwitter's public profile endpoint (X's own API needs a
// paid key). The site build reads that cache and makes no network request.
import { createHash } from 'node:crypto';
import { readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const avatarSize = '_200x200';
const timeoutMs = 20_000;
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const output = path.join(root, 'apps/site/content/testimonials');
const quotes = path.join(output, 'testimonials.json');
const HANDLE = /^[A-Za-z0-9_]{1,15}$/;
const headers = { 'User-Agent': 'arion-update-testimonials' };

async function fetchOK(url) {
  let response;
  try {
    response = await fetch(url, { headers, signal: AbortSignal.timeout(timeoutMs) });
  } catch (error) {
    throw new Error(`Fetching ${url} failed: ${error.message}`);
  }
  if (response.ok) return response;
  throw new Error(`Fetching ${url} failed: HTTP ${response.status}`);
}

const entries = JSON.parse(readFileSync(quotes, 'utf8'));
if (!Array.isArray(entries) || entries.length === 0) throw new Error(`${quotes}: expected a non-empty array`);

const extensions = { 'image/png': 'png', 'image/jpeg': 'jpg', 'image/gif': 'gif', 'image/webp': 'webp' };
const downloaded = [];
for (const { handle } of entries) {
  if (typeof handle !== 'string' || !HANDLE.test(handle)) throw new Error(`"${handle}" is not an X username`);
  const response = await fetchOK(`https://api.fxtwitter.com/${encodeURIComponent(handle)}`);
  const { user } = await response.json();
  if (!user?.screen_name || !user.avatar_url) throw new Error(`@${handle}: X has no such account`);
  if (user.screen_name.toLowerCase() !== handle.toLowerCase()) {
    throw new Error(`@${handle} is now @${user.screen_name}; update testimonials.json`);
  }
  // X serves several sizes of one avatar under the same name.
  const image = await fetchOK(user.avatar_url.replace(/_normal(\.\w+)$/, `${avatarSize}$1`));
  const type = image.headers.get('content-type')?.split(';')[0]?.trim();
  const extension = extensions[type];
  if (!extension) throw new Error(`@${handle}: unsupported avatar type ${type}`);
  const bytes = Buffer.from(await image.arrayBuffer());
  downloaded.push({
    handle: user.screen_name,
    name: user.name?.trim() || user.screen_name,
    avatar: `x-${user.screen_name.toLowerCase()}.${extension}`,
    avatarSHA256: createHash('sha256').update(bytes).digest('hex'),
    bytes,
  });
}

for (const name of readdirSync(output)) {
  if (name.startsWith('x-')) rmSync(path.join(output, name));
}
for (const entry of downloaded) writeFileSync(path.join(output, entry.avatar), entry.bytes);
writeFileSync(
  path.join(output, 'profiles.json'),
  `${JSON.stringify(
    {
      source: 'https://api.fxtwitter.com/{handle}',
      avatarSize,
      profiles: downloaded.map(({ bytes, ...entry }) => entry),
    },
    null,
    2,
  )}\n`,
);
console.log(`Wrote ${downloaded.length} testimonial profiles to ${path.relative(root, output)}`);
