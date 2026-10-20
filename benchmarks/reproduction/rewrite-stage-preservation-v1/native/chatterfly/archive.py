"""Archive completed native fixture observations; retain exact original sources."""
import hashlib,json,pathlib,shutil,subprocess,sys
root=pathlib.Path.cwd(); src=pathlib.Path(sys.argv[1]); dst=pathlib.Path(sys.argv[2]);dst.mkdir(parents=True,exist_ok=False)
for name in ['corpus.json','audio','images','render.swift','ocr.jsonl','ocr.stdout.log','ocr.stderr.log','asr.jsonl','asr.stdout.log','asr.stderr.log']:
    p=src/name
    if p.is_dir():shutil.copytree(p,dst/name)
    else:shutil.copy2(p,dst/name)
shutil.copy2(root/'tools/evaluation/live/build-live-audio.py',dst/'build-live-audio.py')
shutil.copy2(__file__,dst/'archive.py')
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
c=json.loads((src/'corpus.json').read_text());a=[json.loads(x) for x in (src/'asr.jsonl').read_text().splitlines()];o={x['caseId']:x for x in map(json.loads,(src/'ocr.jsonl').read_text().splitlines()) if x['level']=='accurate'}
assert len(a)==len(c['samples']);assert all(not x.get('error') for x in a)
terms={'native-01':['许婧','阿尔法工作室'],'native-02':['程语楠','德尔塔项目'],'native-03':['张巍','贝塔科技'],'native-04':['优图','项目阿尔法'],'native-05':['许婧','程语楠','德尔塔项目','阿尔法工作室'],'native-06':['getUserProfile'],'native-07':['张巍','张伟','贝塔科技'],'native-08':['程语楠','优图','阿尔法工作室']}
pairs=[]
for r in a:
    i=r['caseId']; screen=o[i]; assert not screen.get('error')
    pairs.append(dict(id=i,scene='chat',preview=r['events'][-1]['text'],final=r['sourceText'],screen=screen['text'],pairKind='native-asr-vision-synthetic-fixture',checks=terms.get(i,['Chatterfly']),nativeEvidence=dict(audio=str(dst/'audio'/f'{i}.wav'),audioSha256=sha(dst/'audio'/f'{i}.wav'),asrObservations=str(dst/'asr.jsonl'),asrSha256=sha(dst/'asr.jsonl'),ocrObservations=str(dst/'ocr.jsonl'),ocrSha256=sha(dst/'ocr.jsonl'),previewSource='Last observed production streaming ASR DraftReplaced event; not an LLM preview and not proof of closed-segment stability.',finalSource='Production coordinator sourceText after whole-utterance offline finalization and rule correction.',limitations='Synthetic TTS and rendered image, not live user window capture, natural voice, microphone, LLM trace or insertion evidence.')))
(dst/'pairs.json').write_text(json.dumps(pairs,ensure_ascii=False,indent=2))
metadata={'kind':'completed-native-asr-vision-fixtures','gitRevision':subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),'dirtyState':subprocess.check_output(['git','status','--short'],text=True),'platform':subprocess.check_output(['sw_vers'],text=True),'hardware':subprocess.check_output(['sysctl','-n','machdep.cpu.brand_string','hw.memsize'],text=True),'goVersion':subprocess.check_output(['go','version'],text=True),'power':subprocess.check_output(['pmset','-g'],text=True),'treatments':{'bundle':'xasr-160-xasr-final','segmentFinalize':False,'removeFillers':True,'threads':4,'chunkMs':20,'llm':False,'ocr':['accurate','fast']},'commands':{'ocr':f'arion-ocr -corpus {src}/corpus.json -out {src}/ocr.jsonl','asr':f'caffeinate -i arion-live-bench -manifest {src}/audio/manifest.json -corpus {src}/corpus.json -models .workspace/models/store -bundle xasr-160-xasr-final -variant native-no-llm-xasr160 -out {src}/asr.jsonl'},'provenance':{'text':'Developer-authored synthetic text; no personal content. Supplemental Chatterfly term derives from user-reported failure.', 'audio':'macOS say Tingting synthetic voice; generated locally using installed system voice. This records generation provenance, not a claim of ownership or an independent redistribution license for Apple voice assets. No system voice assets included.','image':'Developer-authored synthetic text rendered by AppKit; no captured user desktop content.','limitations':'Synthetic fixtures are diagnostic, not held-out evaluation. Streaming ASR observations are not LLM previews.'},'failedAttempts':[{'stage':'render-source-preparation','status':'failed','error':'Initial command used relative .workspace source path from apps/desktop; source file absent. Retried correct repository-root command; no inference outputs overwritten.'}]}
for name in ['arion-live-bench','arion-ocr']:
    binary=root/'.workspace/rewrite-stage-preservation/native'/name
    metadata.setdefault('binaryHashes',{})[name]=sha(binary)
for model in ['x-asr-zh-en-160ms-streaming-punct-int8-2026-06-05','x-asr-zh-en-offline-punct-int8-2026-06-03']:
    p=root/'.workspace/models/store'/model/'.verified.json'
    metadata.setdefault('verifiedModelManifests',{})[model]=json.loads(p.read_text())
metadata['pathMappings']=[{'original':str(src/p.relative_to(dst)),'archived':str(p),'sha256':sha(p)} for p in sorted(dst.rglob('*')) if p.is_file()]
(dst/'provenance.json').write_text(json.dumps(metadata,ensure_ascii=False,indent=2))
print(json.dumps(pairs,ensure_ascii=False,indent=2))
