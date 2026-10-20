"""Reuse frozen staged payload construction with the explicit v2 recipe."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);result=importlib.util.module_from_spec(spec);spec.loader.exec_module(result);return result
candidates=module('bounded_v2_candidates',HERE/'candidates.py')
core=module('bounded_v1_payload_builder',HERE.parent/'rewrite-bounded-edit-v1/pipeline.py')
core.HERE=HERE;core.candidates=candidates
read=core.read
cleaned=core.cleaned
selection_data=core.selection_data
selection_payload=core.selection_payload
format_data=core.format_data
format_payload=core.format_payload
