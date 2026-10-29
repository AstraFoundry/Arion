import { readFileSync } from "node:fs";
import path from "node:path";
import { authorAvatars, getPost, postSlugs } from "@/entities/post";
import { LOCALES } from "@/shared/i18n";
import { PROJECT_ASSET_BASE, PROJECT_ASSET_TYPES } from "@/shared/lib/server";
import { postMediaRefs } from "@/widgets/post-article";

// Repository files that blog posts name by a project path, and their authors'
// GitHub avatars, written into the static export at
// /blog-assets/<content hash>/<name>. Built once, never served live.
export const dynamic = "force-static";
export const dynamicParams = false;

interface AssetParams {
  hash: string;
  name: string;
}

/** Every project file and author avatar any post names, by its published URL. */
async function blogAssets(): Promise<Map<string, () => Uint8Array>> {
  const assets = new Map<string, () => Uint8Array>();
  for (const slug of postSlugs()) {
    for (const locale of LOCALES) {
      const post = getPost(slug, locale);
      const refs = await postMediaRefs(post.body, post.file);
      if (post.image) refs.push({ url: post.image.src, file: post.image.file });
      for (const ref of refs) {
        const { file } = ref;
        if (file && ref.url.startsWith(PROJECT_ASSET_BASE)) assets.set(ref.url, () => readFileSync(file));
      }
      for (const [url, bytes] of await authorAvatars(post.authors)) assets.set(url, () => bytes);
    }
  }
  return assets;
}

export async function generateStaticParams(): Promise<AssetParams[]> {
  const assets = await blogAssets();
  if (assets.size === 0) {
    // A static export cannot emit a dynamic route with no pages.
    throw new Error(
      "No blog post names a file by project path or has authors, so app/blog-assets has nothing to export; " +
        "reference at least one image by a path relative to its post (docs/frontend/blog.md)",
    );
  }
  return [...assets.keys()].map((url) => {
    const [hash, name] = url.slice(PROJECT_ASSET_BASE.length).split("/");
    return { hash, name };
  });
}

export async function GET(_request: Request, { params }: { params: Promise<AssetParams> }) {
  const { hash, name } = await params;
  const bytes = (await blogAssets()).get(`${PROJECT_ASSET_BASE}${hash}/${name}`);
  if (!bytes) throw new Error(`No blog asset ${hash}/${name}`);
  return new Response(new Uint8Array(bytes()), {
    headers: { "Content-Type": PROJECT_ASSET_TYPES[path.extname(name).toLowerCase()] },
  });
}
