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
    id: "misty-hillside",
    title: "A field of pink flowers on a misty hillside",
    author: "bahareh moradian",
    sourceUrl: "https://unsplash.com/photos/a-field-of-pink-flowers-on-a-misty-hillside-WimA_ve2yA4",
    license: "Unsplash License",
    licenseUrl: "https://unsplash.com/license",
  },
  {
    id: "coastal-blur",
    title: "Rolling coastal landscape with blurred grass",
    author: "T (@tanyabarrow)",
    sourceUrl: "https://unsplash.com/photos/rolling-coastal-landscape-with-blurred-grass-Zy7PAQgRswQ",
    license: "Unsplash License",
    licenseUrl: "https://unsplash.com/license",
  },
  {
    id: "pink-motion",
    title: "Pink flowers with motion blur effect",
    author: "Raphael Nicolas",
    sourceUrl: "https://unsplash.com/photos/pink-flowers-with-motion-blur-effect-DjZ5xwr1cq4",
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
