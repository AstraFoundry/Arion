"""Compute text-derived English identifier hypotheses without modifying source or OCR."""
import itertools,pathlib,re,math
ROOT=pathlib.Path(__file__).resolve().parents[3]
DICTIONARY=ROOT/'benchmarks/artifacts/cmudict-pronunciation-v1/cmudict.dict'
QUOTED=re.compile(r'“[^”]*(?:”|$)|「[^」]*(?:」|$)|『[^』]*(?:』|$)|‘[^’]*(?:’|$)|"[^"]*(?:"|$)|`[^`]*(?:`|$)|(?<!\w)\x27[^\x27]*(?:\x27|$)')
SPOKEN=re.compile(r'[A-Za-z]+(?:[ \t]+[A-Za-z]+)+')
IDENTIFIER=re.compile(r'\b[A-Za-z_][A-Za-z_0-9]*\b')
PARTS=re.compile(r'[A-Z]+(?=[A-Z][a-z]|$)|[A-Z]?[a-z]+|[0-9]+')
class Lexicon:
 def __init__(self):
  self.entries={}
  for line in DICTIONARY.read_text().splitlines():
   fields=line.split('#',1)[0].split()
   if len(fields)<2:continue
   word=re.sub(r'\([0-9]+\)$','',fields[0]).lower();phones=tuple(re.sub(r'[0-9]','',p) for p in fields[1:]);self.entries.setdefault(word,set()).add(phones)
 def pronunciations(self,words):
  choices=[self.entries.get(w.lower(),set()) for w in words]
  if not choices or any(not p for p in choices):return set(),'unknown_word'
  if math.prod(len(p) for p in choices)>4096:return set(),'too_many_pronunciations'
  return {sum(p,()) for p in itertools.product(*choices)},None
 def align(self,source,screen,scene):
  if scene not in ['work','prompt']:return []
  quoted=[m.span() for m in QUOTED.finditer(source)];identifiers=sorted(set(IDENTIFIER.findall(screen)));candidates=[]
  for identifier in identifiers:
   words=PARTS.findall(identifier)
   if len(words)<2 or ''.join(words)!=identifier:continue
   pronunciations,error=self.pronunciations(words)
   if not error:candidates.append((identifier,pronunciations))
  result=[]
  for span in SPOKEN.finditer(source):
   start,end=span.span()
   if any(start<qend and end>qstart for qstart,qend in quoted):continue
   phones,error=self.pronunciations(span.group().split());matches=sorted(name for name,p in candidates if phones.intersection(p))
   status='unknown_pronunciation' if error else 'ambiguous' if len(matches)>1 else 'unique' if matches else 'no_matching_identifier'
   result.append(dict(sourceSpan=span.group(),startUtf8=len(source[:start].encode()),endUtf8=len(source[:end].encode()),status=status,candidateSpellings=matches,provenance='text-derived CMU English pronunciations; no independent acoustic observation',limitation=error))
  return result
