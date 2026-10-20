"""Separate complete untrusted context from source turns without changing content."""
import importlib.util
import json
import pathlib

HERE = pathlib.Path(__file__).resolve().parent
ARMS = ['baseline', 'reference-turn', 'balanced', 'reference-turn-balanced']
SEPARATED = {'reference-turn', 'reference-turn-balanced'}


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


prior = module('reference_turn_prior', HERE.parent/'rewrite-selector-screen-v1/pipeline.py')


def candidates_for(arm):
    if arm not in ARMS:
        raise ValueError('Unknown reference-turn arm')
    return prior.candidates_for('baseline')


def separate(request):
    result = json.loads(json.dumps(request, ensure_ascii=False))
    messages = []
    for message in result['messages']:
        if message['role'] == 'user':
            data = json.loads(message['content'])
            context = data.pop('context')
            messages.append(dict(role='tool', content=json.dumps(dict(context=context), ensure_ascii=False, separators=(',', ':'))))
            message['content'] = json.dumps(data, ensure_ascii=False, separators=(',', ':'))
        messages.append(message)
    result['messages'] = messages
    return result


def restore(request):
    result = json.loads(json.dumps(request, ensure_ascii=False))
    messages = []
    pending = None
    for message in result['messages']:
        if message['role'] == 'tool':
            if pending is not None:
                raise ValueError('Unconsumed context reference')
            reference = json.loads(message['content'])
            if set(reference) != {'context'}:
                raise ValueError('Unexpected context reference fields')
            pending = reference
            continue
        if message['role'] == 'user':
            if pending is None:
                raise ValueError('Missing context reference')
            data = json.loads(message['content'])
            source = data.pop('source')
            message['content'] = json.dumps(dict(source=source, context=pending['context'], **data), ensure_ascii=False, separators=(',', ':'))
            pending = None
        elif pending is not None:
            raise ValueError('Reference does not precede source')
        messages.append(message)
    if pending is not None:
        raise ValueError('Trailing unconsumed context')
    result['messages'] = messages
    return result


def selection_payload(arm, sample, cleanup):
    candidates_for(arm)
    control = 'balanced' if arm in {'balanced', 'reference-turn-balanced'} else 'baseline'
    request, prepared = prior.selection_payload(control, sample, cleanup)
    return (separate(request) if arm in SEPARATED else request), prepared


def template_identity(arm, stage, scene):
    candidates_for(arm)
    if stage != 'selection':
        raise ValueError('Reference-turn study has no formatter')
    if arm not in SEPARATED:
        return prior.template_identity(arm, stage, scene)
    return dict(version='selector-reference-turn-v1-'+arm, path='benchmarks/reproduction/rewrite-selector-reference-turn-v1/layout.json')


def baseline_projection(request, arm):
    candidates_for(arm)
    result = restore(request) if arm in SEPARATED else json.loads(json.dumps(request, ensure_ascii=False))
    return prior.baseline_projection(result, 'balanced') if arm in {'balanced', 'reference-turn-balanced'} else result
