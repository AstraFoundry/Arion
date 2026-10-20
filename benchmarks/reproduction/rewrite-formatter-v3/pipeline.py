"""Reuse source-visible v2 formatting with a separately frozen prompt recipe."""
import pathlib,importlib.util,json
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('formatter_v3_visible_pipeline',HERE.parent/'rewrite-bounded-edit-v2/pipeline.py');prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior)
prior.core.HERE=HERE
candidates=prior.candidates
format_payload=prior.format_payload
def read(name):return json.loads((HERE/name).read_text())
