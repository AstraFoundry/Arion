"""Compose existing selector/formatter with actual current Go cleanup results."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
selector=module('metal_chain_selector',HERE.parent/'rewrite-bounded-edit-v2/pipeline.py')
formatter=module('metal_chain_formatter',HERE.parent/'rewrite-language-profile-v2/pipeline.py')
selector.core.cleaned=lambda sample:sample['speechCleanup']
candidates=selector.candidates
format_payload=formatter.format_payload
def selection_payload(sample,cleanup):return selector.selection_payload(dict(sample,speechCleanup=cleanup))
