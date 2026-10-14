import { LocaleRootLayout, rootMetadata, rootViewport } from "@/app";
import { NotFoundPage, notFoundMetadata } from "@/pages/not-found";

// Served for every unknown URL (`404.html`). It renders outside the route
// layouts, so it brings its own document shell, in the default locale.
export const metadata = { ...rootMetadata("en"), ...notFoundMetadata("en") };
export const viewport = rootViewport;

export default function GlobalNotFound() {
  return (
    <LocaleRootLayout locale="en">
      <NotFoundPage />
    </LocaleRootLayout>
  );
}
