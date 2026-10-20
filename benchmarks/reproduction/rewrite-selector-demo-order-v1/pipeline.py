"""Reorder existing demonstration pairs without changing their content or request data."""
import importlib.util,json,pathlib
HERE=pathlib.Path(__file__).resolve().parent
ARMS=['baseline','negative-last','balanced','negative-last-balanced']
AUGMENTED={'negative-last','negative-last-balanced'}
spec=importlib.util.spec_from_file_location('demo_order_prior',HERE.parent/'rewrite-selector-screen-v1/pipeline.py')
prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior)
def control_arm(arm):
    if arm not in ARMS:raise ValueError('Unknown demonstration-order arm')
    return 'balanced' if arm in {'balanced','negative-last-balanced'} else 'baseline'
def candidates_for(arm):
    control_arm(arm);return prior.candidates_for('baseline')
def reorder(request,*,restore=False):
    value=json.loads(json.dumps(request,ensure_ascii=False));messages=value['messages']
    if len(messages)!=22 or messages[0]['role']!='system' or messages[-1]['role']!='user':raise ValueError('Changed original ten-example message shape')
    blocks=[messages[i:i+2] for i in range(1,21,2)]
    if any([m['role'] for m in block]!=['user','assistant'] for block in blocks):raise ValueError('Changed original demonstration role boundary')
    order=json.loads((HERE/'layout.json').read_text())['pairPermutation']
    if sorted(order)!=list(range(10)):raise ValueError('Incomplete demonstration permutation')
    positions=[order.index(i) for i in range(10)] if restore else order
    value['messages']=[messages[0]]+[m for i in positions for m in blocks[i]]+[messages[-1]]
    return value
def restore_request(request,arm):
    control_arm(arm);return reorder(request,restore=True) if arm in AUGMENTED else json.loads(json.dumps(request,ensure_ascii=False))
def selection_payload(arm,sample,cleanup):
    request,prepared=prior.selection_payload(control_arm(arm),sample,cleanup)
    if arm in AUGMENTED:request=reorder(request)
    return request,prepared
def template_identity(arm,stage,scene):
    control=control_arm(arm)
    if stage!='selection':raise ValueError('Demonstration-order study has no formatter')
    if arm not in AUGMENTED:return prior.template_identity(control,stage,scene)
    return dict(version='selector-demo-order-v1-'+arm,path='benchmarks/reproduction/rewrite-selector-demo-order-v1/layout.json')
def baseline_projection(request,arm):return prior.baseline_projection(restore_request(request,arm),control_arm(arm))
