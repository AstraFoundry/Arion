// Refreshes the contributors shown in Settings › About from GitHub: the
// repository's contributors in GitHub's order (most contributions first),
// without bots, and each one's avatar. The app embeds the result and makes
// no network request for it.
import { createHash } from 'node:crypto';
import { mkdirSync, readdirSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const repository = 'AstraFoundry/Arion';
const avatarSize = 96;
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const output = path.join(root, 'apps/desktop/ui/shared/assets/contributors');

const headers = { Accept: 'application/vnd.github+json', 'User-Agent': 'arion-update-contributors' };
if (process.env.GITHUB_TOKEN) headers.Authorization = `Bearer ${process.env.GITHUB_TOKEN}`;

async function fetchOK(url, options) {
  const response = await fetch(url, options);
  if (!response.ok) throw new Error(`${url}: ${response.status} ${response.statusText}`);
  return response;
}

const contributors = [];
for (let page = 1; ; page++) {
  const batch = await (await fetchOK(`https://api.github.com/repos/${repository}/contributors?per_page=100&page=${page}`, { headers })).json();
  contributors.push(...batch.filter(c => c.type !== 'Bot'));
  if (batch.length < 100) break;
}
if (contributors.length === 0) throw new Error(`${repository} lists no contributors`);

const extensions = { 'image/png': 'png', 'image/jpeg': 'jpg' };
mkdirSync(output, { recursive: true });
for (const name of readdirSync(output)) {
  if (name !== 'contributors.json') rmSync(path.join(output, name));
}
const entries = [];
for (const c of contributors) {
  const avatarURL = new URL(c.avatar_url);
  avatarURL.searchParams.set('s', String(avatarSize));
  const response = await fetchOK(avatarURL, { headers: { 'User-Agent': headers['User-Agent'] } });
  const type = response.headers.get('content-type')?.split(';')[0];
  const extension = extensions[type];
  if (!extension) throw new Error(`${c.login}: unsupported avatar type ${type}`);
  const bytes = Buffer.from(await response.arrayBuffer());
  const avatar = `${c.login}.${extension}`;
  writeFileSync(path.join(output, avatar), bytes);
  entries.push({
    login: c.login,
    profile: c.html_url,
    contributions: c.contributions,
    avatar,
    avatarSHA256: createHash('sha256').update(bytes).digest('hex'),
  });
}
writeFileSync(path.join(output, 'contributors.json'), `${JSON.stringify({ repository, contributors: entries }, null, 2)}\n`);
console.log(`Wrote ${entries.length} contributors to ${path.relative(root, output)}`);
