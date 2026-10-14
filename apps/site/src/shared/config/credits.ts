/**
 * Third-party work shown or shipped on the site. Titles and names are proper
 * nouns, identical in every locale; the credits page supplies the labels.
 */
export interface Credit {
  id: string;
  title: string;
  author: string;
  sourceUrl: string;
  license: string;
  licenseUrl: string;
}

/** Photographs behind the product illustrations (public/backdrops). */
export const photoCredits: readonly Credit[] = [
  {
    id: "flowers-falling",
    title: "Flowers falling",
    author: "Roma Kaiuk",
    sourceUrl: "https://unsplash.com/photos/a-group-of-pink-flowers-z51SGO5m5ec",
    license: "Unsplash License",
    licenseUrl: "https://unsplash.com/license",
  },
];

/** Typefaces, icons and motion components the site ships. */
export const softwareCredits: readonly Credit[] = [
  {
    id: "fraunces",
    title: "Fraunces",
    author: "Undercase Type",
    sourceUrl: "https://github.com/undercasetype/Fraunces",
    license: "SIL Open Font License 1.1",
    licenseUrl: "https://openfontlicense.org",
  },
  {
    id: "ibm-plex-sans",
    title: "IBM Plex Sans",
    author: "IBM",
    sourceUrl: "https://github.com/IBM/plex",
    license: "SIL Open Font License 1.1",
    licenseUrl: "https://openfontlicense.org",
  },
  {
    id: "jetbrains-mono",
    title: "JetBrains Mono",
    author: "JetBrains",
    sourceUrl: "https://github.com/JetBrains/JetBrainsMono",
    license: "SIL Open Font License 1.1",
    licenseUrl: "https://openfontlicense.org",
  },
  {
    id: "lucide",
    title: "Lucide",
    author: "Lucide Contributors",
    sourceUrl: "https://lucide.dev",
    license: "ISC License",
    licenseUrl: "https://lucide.dev/license",
  },
  {
    id: "beui",
    title: "beUI",
    author: "Saurabh Chauhan",
    sourceUrl: "https://beui.dev",
    license: "MIT License",
    licenseUrl: "https://github.com/starc007/ui-components/blob/main/LICENSE",
  },
];
