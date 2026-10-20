"""Compare a pinned small model on unchanged, previously executed selector requests."""
import argparse
import importlib.util
import json
import pathlib
import sys

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))
import gguf_rewrite as engine
from gguf_session_v2 import Session
from gguf_template_identity import identity

spec = importlib.util.spec_from_file_location('model_screen_pipeline', HERE.parent / 'rewrite-selector-screen-v1/pipeline.py')
pipeline = importlib.util.module_from_spec(spec)
spec.loader.exec_module(pipeline)
ROOT = engine.ROOT
ARMS = ['baseline', 'balanced']
PREPS = ROOT / 'benchmarks/runs/rewrite-selector-screen-v1/run/preparations.jsonl'
REFERENCE = ROOT / 'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl'


def rowhash(value):
    import hashlib
    return hashlib.sha256(json.dumps(value, ensure_ascii=False, sort_keys=True).encode()).hexdigest()


def execute():
    parser = argparse.ArgumentParser()
    parser.add_argument('--out', type=pathlib.Path, required=True)
    args = parser.parse_args()
    freeze = engine.read(HERE / 'freeze.json')
    engine.MODEL = ROOT / freeze['modelPath']
    engine.verify(freeze)
    preps = [json.loads(line) for line in PREPS.read_text().splitlines()]
    references = {(r['arm'], r['id']): r for r in map(json.loads, REFERENCE.read_text().splitlines())}
    # Replay all source preparation and compare the entire model request before starting inference.
    for arm in ARMS:
        for prep in preps:
            request, prepared = pipeline.selection_payload(arm, prep['sample'], prep['cleanupObservation']['cleanup'])
            if prepared != prep['prepared']:
                raise ValueError('Changed frozen preparation')
            if prepared['hypotheses'] and request != references[(arm, prep['sample']['id'])]['request']:
                raise ValueError('Changed actual comparison request')
    out = args.out.resolve()
    out.mkdir(parents=True, exist_ok=False)
    manifest = dict(pipeline='selector-model-v1', status='running', arms=ARMS,
                    model=freeze['model'], runtimeFiles=freeze['runtimeFiles'],
                    freezeSha256=engine.sha(HERE / 'freeze.json'), workers=[],
                    replayedSourcePreparations=457, newGoPreparations=0,
                    newAsrGenerations=0, newOcrGenerations=0, formatterGenerations=0,
                    preparationReference=str(PREPS.relative_to(ROOT)), preparationsSha256=engine.sha(PREPS),
                    sourceTreatmentCount=914, expectedGenerations=202)
    engine.write(out / 'manifest.json', manifest)
    worker = None
    count = skipped = errors = 0
    try:
        with (out / 'results.jsonl').open('x') as stream, (out / 'case-outcomes.jsonl').open('x') as outcomes:
            for arm in ARMS:
                for scene in ['work', 'email', 'prompt', 'chat']:
                    subset = [p for p in preps if p['sample']['scene'] == scene]
                    if any(p['prepared']['hypotheses'] for p in subset):
                        name = 'selector-model-v1-' + arm + '-' + scene
                        worker = Session(out, name, 18823, gpu_layers=99, verbosity=5)
                        manifest['workers'].append(dict(arm=arm, scene=scene, name=name, command=worker.command, startupMs=worker.startup_ms))
                    for prep in subset:
                        sample = prep['sample']
                        outcome = dict(id=sample['id'], arm=arm, preparationRowSha256=rowhash(prep), formatterExecuted=False)
                        if not prep['prepared']['hypotheses']:
                            skipped += 1
                            outcome.update(status='no-hypotheses', modelSelectorCalled=False, resultRowSha256=None)
                        else:
                            request, prepared = pipeline.selection_payload(arm, sample, prep['cleanupObservation']['cleanup'])
                            prompt = identity(ROOT, freeze, pipeline, arm, 'selection', scene, request)
                            response, elapsed, rss = worker.generate(request)
                            raw = response['choices'][0]['message']['content']
                            edited = choice = error = None
                            edits = []
                            try:
                                edited, edits = pipeline.candidates_for(arm).apply(raw, prepared)
                                choice = json.loads(raw)
                            except (ValueError, KeyError, TypeError) as issue:
                                error = type(issue).__name__ + ': ' + str(issue)
                                errors += 1
                            reference = references[(arm, sample['id'])]
                            row = dict(sample, arm=arm, variant=arm+'-selection', pipeline='selector-model-v1', stage='selection',
                                       effectiveModelScene='spelling-selection', evaluationSet=sample['cohort'],
                                       sourceControl=prep['sourceControl'], preparationRowSha256=rowhash(prep),
                                       sourcePreparation=dict(kind='replayed-frozen-actual-go-preparation', modelSelectorCalled=True, formatterExecuted=False),
                                       cleanup=prep['cleanupObservation']['cleanup'], cleanupObservation=prep['cleanupObservation'],
                                       prepared=prepared, actualContextUsed=sample['context'], **prompt,
                                       request=request, response=response, selection=choice, editedSource=edited, edits=edits,
                                       renderedBody=None, renderError=error, modelSha256=freeze['model']['sha256'],
                                       generationMs=elapsed, residentRssBytes=rss,
                                       comparisonReference=dict(path=str(REFERENCE.relative_to(ROOT)), arm=arm, rowSha256=rowhash(reference), sameRequest=True))
                            stream.write(json.dumps(row, ensure_ascii=False)+'\n')
                            stream.flush()
                            count += 1
                            outcome.update(status='selection-invalid' if error else 'selected', modelSelectorCalled=True, resultRowSha256=rowhash(row))
                            print(arm, sample['id'], repr(raw), repr(edited), error, flush=True)
                        outcomes.write(json.dumps(outcome, ensure_ascii=False)+'\n')
                        outcomes.flush()
                    if worker:
                        worker.close()
                        worker = None
        manifest.update(status='complete', generations=count, selectorGenerations=count,
                        skippedSourceTreatments=skipped, stageProtocolFailures=errors,
                        resultsSha256=engine.sha(out/'results.jsonl'), outcomesSha256=engine.sha(out/'case-outcomes.jsonl'))
        engine.write(out/'manifest.json', manifest)
    except Exception as issue:
        if worker:
            worker.abort()
        manifest.update(status='failed', completedGenerations=count, error=str(issue))
        engine.write(out/'manifest.json', manifest)
        raise


if __name__ == '__main__':
    execute()
