import "server-only";
import { assetUrl, PROJECT_ASSET_TYPES } from "@/shared/lib/server";
import type { PostAuthor } from "../model/post";

const GITHUB_API = "https://api.github.com";
/** Avatars show at up to 32 CSS pixels; twice that stays sharp on dense screens. */
const AVATAR_PX = 64;
/** How long a build waits for one GitHub response. */
const TIMEOUT_MS = 20_000;

/** The fields of `GET /users/{login}` the site reads. */
interface GitHubUser {
  login: string;
  name: string | null;
  html_url: string;
  avatar_url: string;
}

interface ResolvedAuthor {
  author: PostAuthor;
  avatar: Uint8Array;
}

/** Image extensions by media type, from the types `/blog-assets/` serves. */
const IMAGE_EXTENSIONS = new Map(
  Object.entries(PROJECT_ASSET_TYPES)
    .filter(([, type]) => type.startsWith("image/"))
    .map(([extension, type]) => [type, extension]),
);

async function get(url: string, headers: HeadersInit = {}): Promise<Response> {
  const response = await fetch(url, { headers, signal: AbortSignal.timeout(TIMEOUT_MS) });
  if (response.ok) return response;
  const limited = response.status === 403 || response.status === 429;
  throw new Error(
    `Fetching ${url} failed: HTTP ${response.status}` +
      (limited ? "; set GITHUB_TOKEN to raise the GitHub API rate limit" : ""),
  );
}

/** `GITHUB_TOKEN`, when set, raises the anonymous limit of 60 API requests an hour. */
function apiHeaders(): HeadersInit {
  const token = process.env.GITHUB_TOKEN?.trim();
  return {
    Accept: "application/vnd.github+json",
    "X-GitHub-Api-Version": "2022-11-28",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
}

async function resolveAuthor(login: string): Promise<ResolvedAuthor> {
  const response = await get(`${GITHUB_API}/users/${encodeURIComponent(login)}`, apiHeaders());
  const user = (await response.json()) as GitHubUser;

  const avatarUrl = new URL(user.avatar_url);
  avatarUrl.searchParams.set("s", String(AVATAR_PX));
  const image = await get(avatarUrl.toString());
  const type = image.headers.get("content-type")?.split(";")[0].trim() ?? "";
  const extension = IMAGE_EXTENSIONS.get(type);
  if (!extension) throw new Error(`avatar has unsupported type "${type}"`);
  const avatar = new Uint8Array(await image.arrayBuffer());

  return {
    author: {
      login: user.login,
      name: user.name?.trim() || user.login,
      profileUrl: user.html_url,
      avatar: assetUrl(avatar, `${user.login.toLowerCase()}${extension}`),
    },
    avatar,
  };
}

/** One request per login per build process; logins are case-insensitive. */
const resolved = new Map<string, Promise<ResolvedAuthor>>();

function resolve(login: string): Promise<ResolvedAuthor> {
  const key = login.toLowerCase();
  let author = resolved.get(key);
  if (!author) {
    author = resolveAuthor(login).catch((error: Error) => {
      throw new Error(`Blog author "${login}": ${error.message}`);
    });
    resolved.set(key, author);
  }
  return author;
}

/**
 * Profiles of a post's authors, read from GitHub at build time. A login
 * GitHub does not know, or an unreachable API, fails the build.
 */
export const getAuthors = (logins: readonly string[]): Promise<PostAuthor[]> =>
  Promise.all(logins.map(async (login) => (await resolve(login)).author));

/** The avatar bytes behind each author's published URL, for the static export. */
export const authorAvatars = async (logins: readonly string[]): Promise<[string, Uint8Array][]> =>
  (await Promise.all(logins.map(resolve))).map(({ author, avatar }) => [author.avatar, avatar]);
