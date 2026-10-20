"""Extend the frozen producer with English subspans and user-aligned letter evidence."""
import importlib.util,pathlib,re
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('bounded_v1_candidate_resource',HERE.parent/'rewrite-bounded-edit-v1/candidates.py');base=importlib.util.module_from_spec(spec);spec.loader.exec_module(base)
WORD=re.compile(r'[A-Za-z]+')
class WindowLexicon:
 def __init__(self):self.lexicon=base.english.Lexicon()
 def align(self,source,screen,scene):
  if scene not in ['work','prompt']:return []
  quoted=[m.span() for m in base.QUOTED.finditer(source)];results=[]
  identifiers=sorted(set(w for w in base.english.IDENTIFIER.findall(screen) if len(base.english.PARTS.findall(w))>=2 and ''.join(base.english.PARTS.findall(w))==w))
  for run in base.english.SPOKEN.finditer(source):
   words=list(WORD.finditer(source,run.start(),run.end()))
   for count in range(2,min(6,len(words))+1):
    for index in range(len(words)-count+1):
     start,end=words[index].start(),words[index+count-1].end();text=source[start:end]
     if any(base.overlap((start,end),q) for q in quoted):continue
     original=self.lexicon.align(text,screen,scene)
     if len(original)!=1:raise ValueError('Unexpected English word-window coverage')
     match=dict(original[0]);same_letters=[w for w in identifiers if w.casefold()==''.join(text.split()).casefold()]
     # Source letters disambiguate canonical spacing/case before homophonic alternatives.
     if same_letters:match.update(candidateSpellings=same_letters,status='unique' if len(same_letters)==1 else 'ambiguous')
     match.update(startUtf8=base.utf8(source,start),endUtf8=base.utf8(source,end),sourceSpan=text)
     results.append(match)
  return results
base.LEXICON=WindowLexicon()
def prepare(source,context,scene):
 p=base.prepare(source,context,scene)
 for h in p['hypotheses']:
  if h['kind']=='english-identifier':h['sourceLetterMatches']=[w for w in h['candidateSpellings'] if w.casefold()==''.join(h['sourceSpan'].split()).casefold()]
 p['scope']='Fixed full character readings and CMU source English2..6-word windows. Case/space-only OCR canonicalization is source-letter evidence, preceding other homophones. Quotes/numbers/exact spellings excluded; unknowns explicit. Full context retained. No independent acoustic truth.'
 return p
schema=base.schema
apply=base.apply
restore=base.restore
def protect(source,prepared,decisions,edits):
 # Context is already isolated; keep source identities visible to the formatter.
 return dict(encodedSource=source,protectedTerms=[],deliveryPolicy='unfiltered',mode='visible-corrected-source',literalPolicy='Quotes, code, names and identifiers must remain exact through model instructions; no output rejection or source substitution.')
