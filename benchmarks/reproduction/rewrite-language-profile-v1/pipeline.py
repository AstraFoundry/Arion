"""Source-derived script profile and explicit language-preserving edit authority."""
import pathlib,importlib.util,json,unicodedata
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('language_profile_prior_pipeline',HERE.parent/'rewrite-formatter-v3/pipeline.py');prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior)
def profile(source):
 names=[unicodedata.name(c,'') for c in source]
 return dict(hasHan=any(n.startswith(('CJK UNIFIED IDEOGRAPH','CJK COMPATIBILITY IDEOGRAPH')) for n in names),hasLatin=any('LATIN' in n for n in names))
def format_data(source,encoding):return dict(source=encoding['encodedSource'],sourceScripts=profile(encoding['encodedSource']))
prior.prior.core.HERE=HERE;prior.prior.core.format_data=format_data
format_payload=prior.format_payload
candidates=prior.candidates
