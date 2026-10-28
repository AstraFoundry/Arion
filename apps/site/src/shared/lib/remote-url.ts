/** An absolute http(s) URL: a file on another site, such as an image host. */
export function isRemoteUrl(src: string): boolean {
  return /^https?:\/\//i.test(src);
}
