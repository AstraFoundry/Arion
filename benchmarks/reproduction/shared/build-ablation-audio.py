import json,pathlib,subprocess,wave,hashlib
out=pathlib.Path('.workspace/ablation-audio');out.mkdir(exist_ok=False)
corpus=json.load(open('.workspace/ablation-corpus.json'))
selected=['zh-day','zh-person','zh-number','zh-repeat','zh-uncertain','zh-email','zh-list','zh-memory','en-day','en-money','en-repeat','en-email','mixed-repair','mixed-code']
samples=[]
for s in corpus['samples']:
 if s['id'] not in selected:continue
 voice='Samantha' if s['language']=='en' else 'Tingting'
 aiff=out/(s['id']+'.aiff');wav=out/(s['id']+'.wav')
 subprocess.run(['say','-v',voice,'-r','175','-o',str(aiff),s['sourceText']],check=True)
 subprocess.run(['ffmpeg','-loglevel','error','-i',str(aiff),'-ar','16000','-ac','1','-c:a','pcm_s16le',str(wav)],check=True)
 with wave.open(str(wav),'rb') as w:duration=w.getnframes()/w.getframerate()*1000
 samples.append(dict(id=s['id'],split='dev',speakerId='synthetic:'+voice,sessionId='authored:'+s['id'],language=s['language'],unit={'zh':'cer','en':'wer','mixed':'mer'}[s['language']],tags=s['tags']+['synthetic-tts'],audio=dict(path=wav.name,sha256=hashlib.sha256(wav.read_bytes()).hexdigest(),durationMs=duration),reference=s['sourceText'],context='',approvedMemory=[]))
original=json.load(open('.workspace/corpus/v1/corpus.json'))
for s in [s for s in original['samples'] if s['split']=='train'][:2]:
 s=dict(s);s['audio']=dict(s['audio']);s['audio']['path']=str((pathlib.Path('.workspace/corpus/v1')/s['audio']['path']).resolve());samples.append(s)
p=out/'corpus.json';p.write_text(json.dumps(dict(version=1,license=original['license']+'; developer-authored synthetic development challenges; no independent annotations',consent=True,samples=samples),ensure_ascii=False,indent=2))
print('Built',len(selected),'synthetic development audio fixtures')
