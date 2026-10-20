"""Use tone-insensitive character readings as unverified spelling hypotheses."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
s=importlib.util.spec_from_file_location('tone_chain_prior_candidates',HERE.parent/'rewrite-bounded-edit-v2/candidates.py');prior=importlib.util.module_from_spec(s);s.loader.exec_module(prior)
exact_reading=prior.base.same_reading
SEGMENTS={char:frozenset(value.rstrip('012345') for value in readings) for char,readings in prior.base.READINGS.items()}
def segmental_match(a,b):return len(a)==len(b) and all(SEGMENTS.get(x,frozenset()).intersection(SEGMENTS.get(y,frozenset())) for x,y in zip(a,b))
prior.base.same_reading=segmental_match
def prepare(source,context,scene):
 result=prior.prepare(source,context,scene)
 for item in result['hypotheses']:
  if item['kind']=='chinese-character-readings':
   item['exactToneCandidates']=[target for target in item['candidateSpellings'] if exact_reading(item['sourceSpan'],target)]
   item['toneInsensitiveOnlyCandidates']=[target for target in item['candidateSpellings'] if target not in item['exactToneCandidates']]
   if item['toneInsensitiveOnlyCandidates']:item['kind']='chinese-segmental-readings'
 result['scope']='Same fixed character readings, source windows, exclusions, schema and maximal-span policy; segmental syllables may match across tones. These are unverified text-derived hypotheses, not acoustic truth or confirmed spelling. English same-letter and ambiguity rules unchanged.'
 return result
schema=prior.schema;apply=prior.apply;protect=prior.protect;restore=prior.restore
