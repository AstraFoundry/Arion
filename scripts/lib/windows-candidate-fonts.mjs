import path from 'node:path';
import { desktopFonts } from './desktop-fonts.mjs';

// Production and native fixtures compile the same immutable font resource.
export function buildCandidateFonts(root, compiler, directory, run) {
  const { directory: fonts } = desktopFonts(root);
  const resource = path.join(directory, 'candidate-fonts.o');
  const windres = path.dirname(compiler) === '.' ? 'windres' : path.join(path.dirname(compiler), 'windres.exe');
  run(windres, ['-I', fonts, '-i', path.join(root, 'apps/desktop/ime/windows/candidate_fonts.rc'), '-O', 'coff', '-o', resource]);
  return resource;
}
