"""Add four novel semantic contrast demonstrations without changing current inputs."""
import importlib.util
import json
import pathlib

HERE = pathlib.Path(__file__).resolve().parent
ARMS = ['baseline', 'contrast-demo', 'balanced', 'contrast-demo-balanced']
AUGMENTED = {'contrast-demo', 'contrast-demo-balanced'}


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


prior = module('contrast_demo_prior', HERE.parent/'rewrite-selector-screen-v1/pipeline.py')


def control_arm(arm):
    if arm not in ARMS:
        raise ValueError('Unknown contrast-demo arm')
    return 'balanced' if arm in {'balanced', 'contrast-demo-balanced'} else 'baseline'


def candidates_for(arm):
    control_arm(arm)
    return prior.candidates_for('baseline')


def demonstrations(arm):
    control = control_arm(arm)
    records = json.loads((HERE/'demonstrations.json').read_text())['records']
    messages = []
    for record in records:
        variant = record['variants'][control]
        messages.extend([variant['prototypeUserMessage'], variant['proposedAssistantDemonstration']])
    return messages


def remove_demos(request, arm):
    result = json.loads(json.dumps(request, ensure_ascii=False))
    expected = demonstrations(arm)
    if result['messages'][-len(expected)-1:-1] != expected:
        raise ValueError('Missing or changed declared contrast demonstrations')
    del result['messages'][-len(expected)-1:-1]
    return result


def selection_payload(arm, sample, cleanup):
    request, prepared = prior.selection_payload(control_arm(arm), sample, cleanup)
    if arm in AUGMENTED:
        request['messages'][-1:-1] = demonstrations(arm)
    return request, prepared


def template_identity(arm, stage, scene):
    control = control_arm(arm)
    if stage != 'selection':
        raise ValueError('Contrast-demo study has no formatter')
    if arm not in AUGMENTED:
        return prior.template_identity(control, stage, scene)
    return dict(version='selector-contrast-demo-v1-'+arm, path='benchmarks/reproduction/rewrite-selector-contrast-demo-v1/layout.json')


def baseline_projection(request, arm):
    control = control_arm(arm)
    result = remove_demos(request, arm) if arm in AUGMENTED else json.loads(json.dumps(request, ensure_ascii=False))
    return prior.baseline_projection(result, control)
