"""Archive completed native v5 evidence, excluding executables and runtime files."""
import hashlib,json,pathlib,shutil
root=pathlib.Path.cwd();src=pathlib.Path(__file__).resolve().parent
out=root/'benchmarks/reproduction/rewrite-stage-preservation-v1/native/final-v5';out.mkdir(exist_ok=False)
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
def rows(path):return [json.loads(line) for line in path.read_text().splitlines()]
def trace(variant):return [r for p in sorted((src/variant).glob('*-rewrite.jsonl')) for r in rows(p)]
def observations(variant):
 result=[]
 for row in rows(src/variant/'asr.jsonl'):
  result.append(dict(caseId=row['caseId'],source=row['sourceText'],final=row['finalText'],raw=row['rawText'],previewEvents=row['previewEvents'],successfulCanonicalPreview=any(e.get('preview') and 'Chatterfly' in e['text'] for e in row['events']),rewrite=row['rewrite'],error=row.get('error'),stopToFinalMs=row['stopToFinalMs']))
 return result
native=observations('native-three');regressions=observations('regressions')
for row in native:row['finalCanonicalNamePresent']='Chatterfly' in row['final']
notes={
 'native-06':dict(outcome='semantic-failure-despite-identifier-repair',identifierRestored=True,reason='Spoken script requests only adding comments. ASR source has ambiguous 之补充注释; final rewrite adds 不要补充注释, reversing that action. This case is not a full pass.',expectedSpeech='请检查 get user profile 这个函数|不要改动函数名只补充注释'),
 'native-07':dict(outcome='ambiguous-name-unresolved-with-abstention',reason='Screen contains both 张伟 and 张巍. Source ASR 张威 remains unchanged; no unsupported choice is made, but the intended name is not recovered.'),
 'native-numeric-01':dict(outcome='numeric-contrast-preserved',reason='3.5 remains the requested version and 3.6 remains excluded.'),
 'native-numeric-02':dict(outcome='numeric-contrast-preserved',reason='一千二百元 remains the budget and 一千五百元 remains excluded.')}
for row in regressions:row.update(notes[row['caseId']])
normalize=lambda req:{k:v for k,v in req.items() if k!='Live'}
cache={}
for label,prefix in [('fiveSecondPilot','cache-'),('tenSecondMatched','cache-extended-')]:
 variants={}
 for treatment in ['on','off']:
  name=prefix+treatment;t=trace(name);variants[treatment]=dict(workerRewriteCalls=len(t),liveCalls=sum(r['Request']['Live'] for r in t),finalCalls=sum(not r['Request']['Live'] for r in t),successfulLiveCalls=sum(r['Request']['Live'] and not r['Error'] for r in t),observations=observations(name),traceFiles=[str(p.relative_to(src)) for p in (src/name).glob('*rewrite.jsonl')])
 cache[label]=variants
cache['fiveSecondPilot']['limitation']='Scheduling-confounded: full live result completed in on only; off full live request cancelled at stop. Do not interpret timing or strict completion-matched benefit.'
on=[r for r in trace('cache-extended-on') if r['Request']['Live'] and not r['Error']][-1]
off=[r for r in trace('cache-extended-off') if r['Request']['Live'] and not r['Error']][-1]
final=[r for r in trace('cache-extended-off') if not r['Request']['Live']][-1]
assert normalize(on['Request'])==normalize(off['Request'])==normalize(final['Request'])
assert on['Result']['Text']==off['Result']['Text']==final['Result']['Text']
assert cache['tenSecondMatched']['on']['finalCalls']==0 and cache['tenSecondMatched']['off']['finalCalls']==1
assert cache['tenSecondMatched']['on']['observations'][0]['final']==cache['tenSecondMatched']['off']['observations'][0]['final']
cache['tenSecondMatched']['proof']=dict(allRequestFieldsExceptLiveEqual=True,rawModelTextEqual=True,finalDeliveryEqual=True,normalizedRequest=normalize(on['Request']),rawModelText=on['Result']['Text'],savedFinalModelCalls=1,limitation='Same synthetic speech plus ten seconds added silence; exposed diagnostic, not a claim about typical stopping behavior or comparative latency.')
build=json.loads((src/'build-provenance.json').read_text());changed=[name for name,digest in build['sources'].items() if sha(root/name)!=digest];assert not changed
summary=dict(status='completed',qualityStatus='known-semantic-failure-retained',promptId='selector-contextual-v5+natural-layout-v3',nativeThree=native,cacheAblation=cache,regressions=regressions,sourceFilesUnchangedDuringRuns=True,limitations=['Synthetic TTS, authored screen PNGs and actual Vision/ASR/LLM. No user microphone, native window capture, editor insertion or Windows evidence.','Original approved memory constructed after initial ASR observations: exposed development reproduction, not held-out quality or preexisting user memory.','Screen-off native01 and cache variants reuse original speech and are not independent samples.','Timings are raw observations. Five-second cache pilot has scheduling confound; ten-second pair proves request reuse without establishing typical latency.','Technical regression reveals added negation; preserve failure rather than declaring entire pipeline correct.'])
(src/'summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2))
mappings=[]
for file in sorted(src.rglob('*')):
 if not file.is_file():continue
 rel=file.relative_to(src)
 if rel.parts[0]=='runtime' or file.name=='arion-live-bench':continue
 target=out/rel;target.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(file,target);assert sha(target)==sha(file)
 mappings.append(dict(path=str(rel),original=str(file.relative_to(root)),sha256=sha(target),bytes=target.stat().st_size))
(out/'README.md').write_text('''# Final v5 native evidence

Actual Vision, production ASR/coordinator and source-built v5 worker run on frozen synthetic fixtures. See [summary.json](summary.json) for exact outcomes and [manifest.json](manifest.json) for original paths and file hashes. Runtime executables and model weights are excluded; their identities remain in build/invocation metadata. Source Markdown is preserved byte-for-byte as `.md.txt` evidence with explicit mappings in build provenance.

All three Chatterfly reproduction finals are correct. Original case 01 and its screen-off duplicate have validated correct previews; case 02 has a cancelled live request and a corrected final. The ten-second-silence cache pair has identical complete effective requests except Live and identical raw output: cache-on performs zero final worker calls, cache-off performs one. The five-second pilot is retained with its scheduling confound.

The technical regression restores getUserProfile but adds “不要补充注释”, reversing the intended positive action after ambiguous ASR “之补充注释”. It is a semantic failure, not a passing pipeline case. Competing-contact disambiguation abstains and leaves the ASR name unresolved. Both numeric contrasts are preserved.

These are exposed development diagnostics. Approved memory was constructed after earlier observations; duplicated speech and authored silence are not independent test samples. No held-out quality, typical latency, native insertion or Windows claim follows.
''')
mappings.append(dict(path='README.md',original='Generated archive evidence guide',sha256=sha(out/'README.md'),bytes=(out/'README.md').stat().st_size))
manifest=dict(status='completed',kind='arion-final-v5-native-evidence',summary='summary.json',sourceSnapshot='source-snapshot',runtimeAndWeightsIncluded=False,sourceFilesUnchangedDuringRuns=True,files=mappings)
(out/'manifest.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2))
for row in mappings:assert sha(out/row['path'])==row['sha256']
print(json.dumps(dict(archive=str(out),verifiedFiles=len(mappings),summary=summary),ensure_ascii=False,indent=2))
