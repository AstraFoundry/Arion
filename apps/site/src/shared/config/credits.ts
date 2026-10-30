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
    id: "pink-motion",
    title: "Pink flowers with motion blur effect",
    author: "Raphael Nicolas",
    sourceUrl: "https://unsplash.com/photos/pink-flowers-with-motion-blur-effect-DjZ5xwr1cq4",
    license: "Unsplash License",
    licenseUrl: "https://unsplash.com/license",
  },
  {
    id: "pink-gladiolus",
    title: "Pink gladiolus flowers with green foliage in motion",
    author: "Eugene Golovesov",
    sourceUrl:
      "https://unsplash.com/photos/pink-gladiolus-flowers-with-green-foliage-in-motion-Jk_9EwoLXnY",
    license: "Unsplash License",
    licenseUrl: "https://unsplash.com/license",
  },
  {
    id: "coneflower-swirl",
    title: "Pink coneflowers with a swirling motion effect",
    author: "Brooke Balentine",
    sourceUrl:
      "https://unsplash.com/photos/pink-coneflowers-with-a-swirling-motion-effect--Ck7iqGR1pc",
    license: "Unsplash License",
    licenseUrl: "https://unsplash.com/license",
  },
  {
    id: "rose-bush",
    title: "Blurred pink flowers in a vibrant, green bush",
    author: "Eugene Golovesov",
    sourceUrl: "https://unsplash.com/photos/blurred-pink-flowers-in-a-vibrant-green-bush-cTChGLob-2k",
    license: "Unsplash License",
    licenseUrl: "https://unsplash.com/license",
  },
  {
    id: "dark-roses",
    title: "Blurred pink flowers against dark background",
    author: "Brigitte Elsner",
    sourceUrl: "https://unsplash.com/photos/blurred-pink-flowers-against-dark-background-_NnGJYw02Z8",
    license: "Unsplash License",
    licenseUrl: "https://unsplash.com/license",
  },
  {
    id: "pink-grass",
    title: "Pink flowers and green grass with motion blur",
    author: "Eugene Golovesov",
    sourceUrl: "https://unsplash.com/photos/pink-flowers-and-green-grass-with-motion-blur-6mV2lNqyLps",
    license: "Unsplash License",
    licenseUrl: "https://unsplash.com/license",
  },
  {
    id: "soft-pink",
    title: "Soft pink flowers with green leaves in soft focus",
    author: "Ruben Mavarez",
    sourceUrl:
      "https://unsplash.com/photos/soft-pink-flowers-with-green-leaves-in-soft-focus-yi0RrmWvd-8",
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
