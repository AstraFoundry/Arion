"""Add concise public referent descriptions while retaining bounded candidate application."""
import importlib.util
import json
import pathlib

HERE = pathlib.Path(__file__).resolve().parent
ARMS = ['baseline', 'referent', 'balanced', 'referent-balanced']
EXPLANATORY = {'referent', 'referent-balanced'}


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


prior = module('referent_prior_screen', HERE.parent/'rewrite-selector-screen-v1/pipeline.py')


def decision_choices(arm, selection, prepared):
    if arm not in ARMS:
        raise ValueError('Unknown referent arm')
    if arm not in EXPLANATORY:
        return selection
    expected = {h['id'] for h in prepared['hypotheses']}
    if not isinstance(selection, dict) or set(selection) != expected:
        raise ValueError('Missing or fabricated explanatory candidate id')
    choices = {}
    for key, record in selection.items():
        if not isinstance(record, dict) or set(record) != {'referent', 'choice'}:
            raise ValueError('Invalid explanatory record')
        if not isinstance(record['referent'], str) or not 1 <= len(record['referent']) <= 48:
            raise ValueError('Invalid public referent description')
        choices[key] = record['choice']
    return choices


class ExplainedCandidates:
    def __init__(self, arm):
        self.arm = arm

    def apply(self, raw, prepared):
        selection = json.loads(raw)
        choices = decision_choices(self.arm, selection, prepared)
        # Existing application checks candidate identities, permitted choices and exact UTF-8 spans.
        return prior.candidates_for('baseline').apply(json.dumps(choices), prepared)


def candidates_for(arm):
    if arm not in ARMS:
        raise ValueError('Unknown referent arm')
    return ExplainedCandidates(arm) if arm in EXPLANATORY else prior.candidates_for(arm)


def selection_payload(arm, sample, cleanup):
    if arm not in ARMS:
        raise ValueError('Unknown referent arm')
    base_arm = 'balanced' if arm in {'balanced', 'referent-balanced'} else 'baseline'
    request, prepared = prior.selection_payload(base_arm, sample, cleanup)
    request['max_tokens'] = 256
    if arm not in EXPLANATORY:
        return request, prepared
    prompt = json.loads((HERE/'prompts.json').read_text())
    request['messages'][0]['content'] = prompt['selectionSystem']
    assistant_messages = [message for message in request['messages'] if message['role'] == 'assistant']
    if len(assistant_messages) != len(prompt['exampleReferents']):
        raise ValueError('Changed explanatory example alignment')
    for message, referents in zip(assistant_messages, prompt['exampleReferents']):
        choices = json.loads(message['content'])
        if len(choices) != len(referents):
            raise ValueError('Changed candidate coverage in explanatory example')
        record = {key: dict(referent=meaning, choice=value) for (key, value), meaning in zip(choices.items(), referents)}
        message['content'] = json.dumps(record, ensure_ascii=False, separators=(',', ':'))
    schema = request['response_format']['schema']
    schema['properties'] = {key: dict(type='object', properties=dict(referent=dict(type='string', minLength=1, maxLength=48), choice=value),
                                    required=['referent', 'choice'], additionalProperties=False)
                            for key, value in schema['properties'].items()}
    return request, prepared


def template_identity(arm, stage, scene):
    if arm not in ARMS or stage != 'selection':
        raise ValueError('Unknown referent prompt stage')
    if arm not in EXPLANATORY:
        return prior.template_identity(arm, stage, scene)
    return dict(version='selector-referent-v1-'+arm, path='benchmarks/reproduction/rewrite-selector-referent-v1/prompts.json')


def baseline_projection(request, arm):
    result = json.loads(json.dumps(request, ensure_ascii=False))
    if arm in EXPLANATORY:
        result['messages'][0]['content'] = json.loads((HERE.parent/'rewrite-selector-screen-v1/templates/baseline/prompts.json').read_text())['selectionSystem']
        for message in result['messages']:
            if message['role'] == 'assistant':
                message['content'] = json.dumps({key: value['choice'] for key, value in json.loads(message['content']).items()}, separators=(',', ':'))
        result['response_format']['schema']['properties'] = {
            key: value['properties']['choice'] for key, value in result['response_format']['schema']['properties'].items()}
    if arm in {'balanced', 'referent-balanced'}:
        result = prior.baseline_projection(result, 'balanced')
    return result
