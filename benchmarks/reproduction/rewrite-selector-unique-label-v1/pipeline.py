"""Omit only unverified unique-status labels from model-facing hypotheses."""
import importlib.util
import json
import pathlib

HERE = pathlib.Path(__file__).resolve().parent
ARMS = ['baseline', 'no-unique-label', 'balanced', 'no-unique-label-balanced']
OMITTED = {'no-unique-label', 'no-unique-label-balanced'}


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


prior = module('unique_label_prior', HERE.parent/'rewrite-selector-screen-v1/pipeline.py')


def candidates_for(arm):
    if arm not in ARMS:
        raise ValueError('Unknown unique-label arm')
    return prior.candidates_for('baseline')


def transform(request, *, omit):
    result = json.loads(json.dumps(request, ensure_ascii=False))
    for message in result['messages']:
        if message['role'] != 'user':
            continue
        data = json.loads(message['content'])
        for index, hypothesis in enumerate(data['hypotheses']):
            if omit:
                if hypothesis['status'] == 'unique':
                    del hypothesis['status']
            elif 'status' not in hypothesis:
                restored = {}
                for key, value in hypothesis.items():
                    if key == 'kind':
                        restored['status'] = 'unique'
                    restored[key] = value
                if 'status' not in restored:
                    raise ValueError('Missing original hypothesis kind')
                data['hypotheses'][index] = restored
        message['content'] = json.dumps(data, ensure_ascii=False, separators=(',', ':'))
    return result


def selection_payload(arm, sample, cleanup):
    candidates_for(arm)
    control = 'balanced' if arm in {'balanced', 'no-unique-label-balanced'} else 'baseline'
    request, prepared = prior.selection_payload(control, sample, cleanup)
    return (transform(request, omit=True) if arm in OMITTED else request), prepared


def template_identity(arm, stage, scene):
    candidates_for(arm)
    if stage != 'selection':
        raise ValueError('Unique-label study has no formatter')
    if arm not in OMITTED:
        return prior.template_identity(arm, stage, scene)
    return dict(version='selector-unique-label-v1-'+arm, path='benchmarks/reproduction/rewrite-selector-unique-label-v1/layout.json')


def baseline_projection(request, arm):
    candidates_for(arm)
    result = transform(request, omit=False) if arm in OMITTED else json.loads(json.dumps(request, ensure_ascii=False))
    return prior.baseline_projection(result, 'balanced') if arm in {'balanced', 'no-unique-label-balanced'} else result
