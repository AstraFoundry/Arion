// Fails when a locale dictionary does not have exactly the keys of English,
// changes a value's type, drops or renames a {placeholder}, or leaves a value
// empty. `pnpm build` runs it first, so an incomplete translation never ships.
import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dir = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../src/shared/i18n/messages",
);
const REFERENCE = "en";
const LOCALES = ["en", "zh", "ja"];

/** Flattens a dictionary to `path → value`; array items are indexed. */
function leaves(value, at = "", out = new Map()) {
  if (typeof value === "string" || typeof value === "number") out.set(at, value);
  else if (Array.isArray(value)) value.forEach((item, i) => leaves(item, `${at}[${i}]`, out));
  else if (value && typeof value === "object") {
    for (const [key, child] of Object.entries(value)) leaves(child, at ? `${at}.${key}` : key, out);
  } else throw new Error(`${at}: unexpected value ${String(value)}`);
  return out;
}

const placeholders = (value) =>
  typeof value === "string" ? [...value.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort().join(",") : "";

const load = async (locale) =>
  leaves(JSON.parse(await readFile(path.join(dir, `${locale}.json`), "utf8")));

const reference = await load(REFERENCE);
const errors = [];
for (const locale of LOCALES) {
  const entries = await load(locale);
  for (const [key, value] of reference) {
    if (!entries.has(key)) {
      errors.push(`${locale}: missing ${key}`);
      continue;
    }
    const translated = entries.get(key);
    if (typeof translated !== typeof value) errors.push(`${locale}: ${key} is a ${typeof translated}`);
    if (placeholders(translated) !== placeholders(value)) {
      errors.push(`${locale}: ${key} placeholders {${placeholders(translated)}} ≠ {${placeholders(value)}}`);
    }
  }
  for (const [key, value] of entries) {
    if (!reference.has(key)) errors.push(`${locale}: ${key} is not in ${REFERENCE}.json`);
    if (typeof value === "string" && (value.trim() === "" || /TODO|lorem ipsum/i.test(value))) {
      errors.push(`${locale}: ${key} is empty or a placeholder`);
    }
  }
}

if (errors.length) {
  for (const error of errors) console.error(error);
  console.error(`Dictionary check failed: ${errors.length} problem(s).`);
  process.exitCode = 1;
} else {
  console.log(`Dictionaries complete: ${reference.size} entries in ${LOCALES.join(", ")}.`);
}
