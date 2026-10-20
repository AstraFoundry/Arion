"""Change only the trusted system instruction language on unchanged selector data."""
import importlib.util
import json
import pathlib
HERE = pathlib.Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('instruction_language_prior', HERE.parent/'rewrite-selector-screen-v1/pipeline.py')
prior = importlib.util.module_from_spec(spec)
spec.loader.exec_module(prior)
ARMS = ['baseline', 'balanced']


def candidates_for(arm):
    if arm not in ARMS:
        raise ValueError('Unknown instruction-language arm')
    return prior.candidates_for(arm)


def selection_payload(arm, sample, cleanup):
    candidates_for(arm)
    request, prepared = prior.selection_payload(arm, sample, cleanup)
    request['messages'][0]['content'] = json.loads((HERE/'prompts.json').read_text())['selectionSystem']
    return request, prepared


def chinese_projection(request, arm):
    candidates_for(arm)
    result = json.loads(json.dumps(request, ensure_ascii=False))
    result['messages'][0]['content'] = json.loads((HERE.parent/'rewrite-selector-screen-v1/templates'/arm/'prompts.json').read_text())['selectionSystem']
    return result


def template_identity(arm, stage, scene):
    candidates_for(arm)
    if stage != 'selection':
        raise ValueError('Instruction-language study has no formatter')
    return dict(version='selector-instruction-language-v1-'+arm, path='benchmarks/reproduction/rewrite-selector-instruction-language-v1/prompts.json')
