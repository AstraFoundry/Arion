// Command-line flags of the Qwen diagnostic entry points: `--name value` pairs.
export function parseFlags(args) {
  return Object.fromEntries(
    Array.from({ length: args.length / 2 }, (_, i) => [
      args[i * 2].replace(/^--/, ""),
      args[i * 2 + 1],
    ]),
  );
}

export function requireFlags(flags, names) {
  for (const name of names)
    if (!flags[name]) throw new Error(`--${name} required`);
}
