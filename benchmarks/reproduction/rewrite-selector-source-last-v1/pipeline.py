"""Move only the top-level source field after all complete reference/candidate data."""
import importlib.util
import json
import pathlib

HERE = pathlib.Path(__file__).resolve().parent
ARMS = ['baseline', 'source-last', 'balanced', 'source-last-balanced']
REORDERED = {'source-last', 'source-last-balanced'}


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


prior = module('source_last_prior_screen', HERE.parent/'rewrite-selector-screen-v1/pipeline.py')


def candidates_for(arm):
    if arm not in ARMS:
        raise ValueError('Unknown source-order arm')
    return prior.candidates_for('baseline')


def reorder_source(request, *, last):
    result = json.loads(json.dumps(request, ensure_ascii=False))
    for message in result['messages']:
        if message['role'] != 'user':
            continue
        data = json.loads(message['content'])
        source = data.pop('source')
        ordered = dict(data, source=source) if last else dict(source=source, **data)
        message['content'] = json.dumps(ordered, ensure_ascii=False, separators=(',', ':'))
    return result


def selection_payload(arm, sample, cleanup):
    candidates_for(arm)
    original_arm = 'balanced' if arm in {'balanced', 'source-last-balanced'} else 'baseline'
    request, prepared = prior.selection_payload(original_arm, sample, cleanup)
    return (reorder_source(request, last=True) if arm in REORDERED else request), prepared


def template_identity(arm, stage, scene):
    candidates_for(arm)
    if stage != 'selection':
        raise ValueError('Source-order study has no formatter')
    if arm not in REORDERED:
        return prior.template_identity(arm, stage, scene)
    return dict(version='selector-source-last-v1-'+arm, path='benchmarks/reproduction/rewrite-selector-source-last-v1/layout.json')


def baseline_projection(request, arm):
    candidates_for(arm)
    result = reorder_source(request, last=False) if arm in REORDERED else json.loads(json.dumps(request, ensure_ascii=False))
    if arm in {'balanced', 'source-last-balanced'}:
        result = prior.baseline_projection(result, 'balanced')
    return result
