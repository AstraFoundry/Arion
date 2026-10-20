"""Reuse the unchanged cache-enabled full-chain payloads."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('primed_cache_payload',HERE.parent/'rewrite-metal-prefix-v1/pipeline.py');prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior)
candidates=prior.candidates;selection_payload=prior.selection_payload;format_payload=prior.format_payload
