"""Export a complete fixed character resource, never a test-specific vocabulary."""
import pathlib,json,hashlib,sys,re
import pypinyin
from pypinyin import pinyin_dict
ROOT=pathlib.Path(__file__).resolve().parents[3];HERE=pathlib.Path(__file__).resolve().parent
if pypinyin.__version__!='0.55.0':raise ValueError('Require pinned pypinyin0.55.0')
if (HERE/'readings.json').exists():raise FileExistsError('Existing fixed resource; verify instead of overwriting')
rows={};unsupported=[]
for codepoint in sorted(pinyin_dict.pinyin_dict):
 char=chr(codepoint);values=pypinyin.pinyin(char,style=pypinyin.Style.TONE3,heteronym=True,neutral_tone_with_five=True,errors='default')[0];accepted=sorted(set(v for v in values if re.fullmatch('[a-zv]+[1-5]',v)))
 if accepted:rows[char]=accepted
 else:unsupported.append(char)
(HERE/'readings.json').write_text(json.dumps(rows,ensure_ascii=False,sort_keys=True,separators=(',',':'))+'\n')
package=pathlib.Path(pypinyin.__file__).parent;dist=package.parent/'pypinyin-0.55.0.dist-info';sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest();sources=[]
for filename,path in [('source-pinyin-dict.json',package/'pinyin_dict.json'),('source-core.py',package/'core.py'),('METADATA',dist/'METADATA'),('LICENSE.txt',dist/'licenses/LICENSE.txt')]:
 (HERE/filename).write_bytes(path.read_bytes());sources.append(dict(originalPath=str(path.relative_to(ROOT)),archivedPath=str((HERE/filename).relative_to(ROOT)),sha256=sha(path)))
provenance=dict(kind='fixed-character-pronunciation-resource',version='pypinyin0.55.0',source='https://github.com/mozillazg/python-pinyin',python=sys.version,characters=len(rows),unsupportedCharacters=unsupported,readingsSha256=sha(HERE/'readings.json'),sources=sources,scope='All builtin character entries exported using TONE3/heteronym/neutral-tone5. No corpus-derived/custom readings. This is character-level text evidence, not independent acoustic evidence or phrase disambiguation. Missing/unsupported characters remain unknown, never guessed.')
(HERE/'provenance.json').write_text(json.dumps(provenance,ensure_ascii=False,indent=2)+'\n');print('Exported fixed character readings:',len(rows),'unsupported:',len(unsupported))
