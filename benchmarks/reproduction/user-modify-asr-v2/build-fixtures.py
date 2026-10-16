"""Freeze new audio before development tuning, keeping v1 as exposed dev data."""
import argparse,hashlib,json,pathlib,shutil,subprocess,wave
import numpy as np

p=argparse.ArgumentParser();p.add_argument('--out',required=True);a=p.parse_args();root=pathlib.Path(a.out);root.mkdir(parents=True,exist_ok=False)
v1=pathlib.Path('benchmarks/corpora/user-modify-asr-v1')
sha=lambda p:hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()

def write(p,d):p.write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')
config=json.loads((v1/'config.json').read_text());config.update(id='user-modify-asr-v2',retrievalLimit=2,fuzzyThreshold=0.25,arms=['none','legacy_pairs','global_destinations','retrieved_exact','retrieved_fuzzy','retrieved_hint'])
for split in ['dev','test']:
 out=root/split;out.mkdir()
 for name in ['memory.json','model-metadata.json','all-model-files.json','warmup.wav','warmup.aiff','warmup.json']:shutil.copy2(v1/name,out/name)
 write(out/'config.json',config)
# All v1 inputs are disclosed development/regression data, never new held-out data.
dev=json.loads((v1/'corpus.json').read_text())
for s in dev['samples']:
 shutil.copy2(v1/s['audio']['path'],root/'dev'/s['audio']['path']);s['split']='dev'
dev['provenance']+=' Reused exposed v1 fixtures as development data for v2.'
write(root/'dev/corpus.json',dev)

cases=[
 ('arion-1','mixed','recurring','今天先测试阿里昂的快捷键。','今天先测试Arion的快捷键。','Arion'),
 ('arion-2','mixed','recurring','阿里昂的麦克风权限还没有打开。','Arion的麦克风权限还没有打开。','Arion'),
 ('mygo-1','mixed','recurring','我们决定用麦狗制作这个面板。','我们决定用MyGo制作这个面板。','MyGo'),
 ('mygo-2','mixed','recurring','麦狗的输入框需要支持中文。','MyGo的输入框需要支持中文。','MyGo'),
 ('redis-1','mixed','recurring','瑞迪斯里的缓存暂时不要删除。','Redis里的缓存暂时不要删除。','Redis'),
 ('redis-2','mixed','recurring','请确认瑞迪斯是否已经连接成功。','请确认Redis是否已经连接成功。','Redis'),
 ('kubernetes-1','mixed','recurring','这个问题只在库伯内蒂斯里出现。','这个问题只在Kubernetes里出现。','Kubernetes'),
 ('kubernetes-2','mixed','recurring','明天检查库伯内蒂斯的资源用量。','明天检查Kubernetes的资源用量。','Kubernetes'),
 ('feishu-1','zh','recurring','飞书里的那份表格已经更新了。','飞书里的那份表格已经更新了。','飞书'),
 ('feishu-2','zh','recurring','请在飞书上提醒大家参加培训。','请在飞书上提醒大家参加培训。','飞书'),
 ('safari-1','en','recurring','Safari should remember this page after a restart.','Safari should remember this page after a restart.','Safari'),
 ('safari-2','mixed','recurring','沙发浏览器的下载已经结束了。','Safari浏览器的下载已经结束了。','Safari'),
 ('openai-1','en','recurring','Open AI has published another technical report.','OpenAI has published another technical report.','OpenAI'),
 ('openai-2','en','recurring','The team will review the Open AI integration.','The team will review the OpenAI integration.','OpenAI'),
 ('sql-1','en','recurring','The sequel database is unavailable at the moment.','The SQL database is unavailable at the moment.','SQL'),
 ('sql-2','en','recurring','Review the sequel statement before running the migration.','Review the SQL statement before running the migration.','SQL'),
 ('github-1','mixed','recurring','吉特哈布上的这个仓库需要增加说明。','GitHub上的这个仓库需要增加说明。','GitHub'),
 ('github-2','mixed','recurring','我已经在吉特哈布提交了修复。','我已经在GitHub提交了修复。','GitHub'),
 ('codex-1','mixed','recurring','让扣代克斯整理一下测试结果。','让Codex整理一下测试结果。','Codex'),
 ('codex-2','mixed','recurring','扣代克斯没有修改任何配置。','Codex没有修改任何配置。','Codex'),
 ('sofa-1','zh','homophone_control','这张沙发放在客厅里刚好合适。','这张沙发放在客厅里刚好合适。','沙发'),
 ('sofa-2','zh','homophone_control','搬家的时候不要刮坏沙发。','搬家的时候不要刮坏沙发。','沙发'),
 ('sequel-1','en','homophone_control','The director is planning a sequel to the original film.','The director is planning a sequel to the original film.','sequel'),
 ('sequel-2','en','homophone_control','I enjoyed the first book more than its sequel.','I enjoyed the first book more than its sequel.','sequel'),
 ('open-1','en','homophone_control','Keep the door open, I will return in a moment.','Keep the door open, I will return in a moment.',''),
 ('open-2','en','homophone_control','We should stay open, I think, to other suggestions.','We should stay open, I think, to other suggestions.',''),
 ('mai-1','zh','homophone_control','那个卖狗粮的店铺周末不开门。','那个卖狗粮的店铺周末不开门。',''),
 ('mai-2','zh','homophone_control','麦沟附近的村民准备修一条新路。','麦沟附近的村民准备修一条新路。','麦沟'),
 ('reverse-1','zh','reverse_control','看书太久会让眼睛感到疲劳。','看书太久会让眼睛感到疲劳。','眼睛'),
 ('reverse-2','zh','reverse_control','我们参观了燕京大学的旧址。','我们参观了燕京大学的旧址。','燕京'),
 ('expired-1','mixed','expired_control','丘德兰特的索引需要重新建立。','Qdrant的索引需要重新建立。','Qdrant'),
 ('expired-2','mixed','expired_control','请比较丘德兰特和其他向量数据库。','请比较Qdrant和其他向量数据库。','Qdrant'),
 ('ordinary-zh-1','zh','ordinary_control','明天早上先去食堂吃饭再去教室。','明天早上先去食堂吃饭再去教室。',''),
 ('ordinary-zh-2','zh','ordinary_control','这个方案可能还需要再讨论一下。','这个方案可能还需要再讨论一下。',''),
 ('ordinary-en-1','en','ordinary_control','Do not send the message until everyone agrees.','Do not send the message until everyone agrees.',''),
 ('ordinary-en-2','en','ordinary_control','The blue folder is probably still on the desk.','The blue folder is probably still on the desk.',''),
]
out=root/'test';samples=[]
for ident,lang,kind,spoken,ref,term in cases:
 voice='Karen' if lang=='en' else 'Eddy (中文（中国大陆）)';aiff=out/(ident+'.aiff');wav=out/(ident+'.wav')
 subprocess.run(['say','-v',voice,'-r','165','-o',str(aiff),spoken],check=True)
 subprocess.run(['ffmpeg','-loglevel','error','-i',str(aiff),'-ar','16000','-ac','1','-c:a','pcm_s16le',str(wav)],check=True)
 with wave.open(str(wav),'rb') as f:duration=f.getnframes()/f.getframerate()*1000
 samples.append(dict(id='fresh-'+ident,split='test',sessionId='new-session-'+ident,speakerId='tts-'+voice,voice=voice,language=lang,unit={'zh':'cer','en':'wer','mixed':'mer'}[lang],kind=kind,reference=ref,spokenText=spoken,term=term,audio=dict(path=wav.name,sha256=sha(wav),durationMs=duration)))
rng=np.random.default_rng(2730)
for ident,pcm in [('silence',np.zeros(32000,dtype='<i2')),('noise',(rng.normal(0,160,32000)).astype('<i2'))]:
 wav=out/(ident+'.wav')
 with wave.open(str(wav),'wb') as f:f.setnchannels(1);f.setsampwidth(2);f.setframerate(16000);f.writeframes(pcm.tobytes())
 samples.append(dict(id='fresh-'+ident,split='test',sessionId='noise-'+ident,speakerId='generated-nonspeech-v2',language='zh',unit='cer',kind='nonspeech',reference='',term='',audio=dict(path=wav.name,sha256=sha(wav),durationMs=2000)))
# Deterministic existing natural-read controls, selected by ID without labels.
base=pathlib.Path('benchmarks/corpora/asr-v1');natural=json.loads((base/'corpus.json').read_text());imports=[]
for language in ['zh','en']:
 chosen=sorted([s for s in natural['samples'] if s['split']=='test' and s['language']==language],key=lambda s:s['id'])[:10]
 for original in chosen:
  s=json.loads(json.dumps(original));src=base/s['audio']['path'];assert sha(src)==s['audio']['sha256'];name='natural-'+src.name;shutil.copy2(src,out/name)
  s.update(id='natural-'+s['id'],kind='natural_control',term='');s['audio']['path']=name;s.pop('approvedMemory',None);samples.append(s)
  imports.append(dict(originalId=original['id'],originalPath=str(src),sha256=sha(src),sourceSpeaker=original['speakerId']))
write(out/'corpus.json',dict(version=2,consent=True,license='Authored text/scripts AGPL-3.0-only; synthetic system-voice audio for research; natural reads retain original source corpus terms.',provenance='New authored test sentences and different TTS voices frozen before v2 development tuning. Natural controls are exposed historical regression read speech, not a new unseen corpus. No independent human annotation. Memory unchanged from v1.',samples=samples,originalNaturalCorpusSha256=sha(base/'corpus.json'),naturalImports=imports,naturalProvenance=natural['provenance'],naturalLicense=natural['license']))
manifest={str(p.relative_to(root)):{'sha256':sha(p),'sizeBytes':p.stat().st_size} for p in root.rglob('*') if p.is_file()};write(root/'freeze.json',dict(status='frozen_before_development_inference',files=manifest))
print('Frozen dev',len(dev['samples']),'test',len(samples),'inputs')
