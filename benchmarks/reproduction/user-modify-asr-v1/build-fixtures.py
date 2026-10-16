"""Create authored prior memory and independent synthetic recurrence audio.

These are developer diagnostics, not natural speech or independently held-out data.
"""
import hashlib
import json
import pathlib
import subprocess
import sys
import wave

out = pathlib.Path(sys.argv[1])
out.mkdir(parents=True, exist_ok=False)

def save(name, value):
    (out / name).write_text(json.dumps(value, ensure_ascii=False, indent=2) + '\n')

# Authored prior-session memory is fixed before test utterances are built.
rows = [
    ('arion', '阿里昂', 'Arion', 100, 1, 0),
    ('mygo', '麦狗', 'MyGo', 100, 1, 0),
    ('redis', '瑞迪斯', 'Redis', 98, 9, 20),
    ('kubernetes', '库伯内蒂斯', 'Kubernetes', 97, 8, 18),
    ('feishu', '飞输', '飞书', 96, 7, 16),
    ('safari', '沙发', 'Safari', 95, 15, 1),
    ('openai', 'open ai', 'OpenAI', 94, 14, 1),
    ('sql', 'sequel', 'SQL', 93, 13, 1),
    ('github', '吉特哈布', 'GitHub', 92, 2, 0),
    ('codex', '扣代克斯', 'Codex', 91, 1, 0),
    ('reverse-a', '燕京', '眼睛', 99, 99, 99),
    ('reverse-b', '眼睛', '燕京', 99, 99, 99),
    ('expired', '丘德兰特', 'Qdrant', 60, 999, 999),
]
memory = [dict(id=i, source=a, target=b, updatedDay=d, useCount=u, recallCount=r,
               approved=True, origin='authored_prior_session', sourceSession='prior-'+i)
          for i,a,b,d,u,r in rows]
save('memory.json', dict(version=1, logicalDay=100, entries=memory,
     provenance='Developer-authored approved prior-session fixtures and counts, not mined personal data. Fixed before current test utterances and never derived from their references.'))
config = dict(version=1, id='user-modify-asr-v1', logicalDay=100, expireDays=30,
              todayQuota=2, recallQuota=3, totalLimit=8, maxChars=512,
              arms=['none', 'destinations', 'pairs', 'tagged_pairs', 'all_pairs'],
              model='mlx-community/Qwen3-ASR-1.7B-8bit',
              revision='a8379a2e2f9e313c9292cdf1af4055ab56d50d55',
              maxTokens=192, temperature=0, seed=1729,
              unknownCompetitorSettings='Quotas, limits, tie-breaks, count-trigger details and serialization are experimental declarations, not recovered Doubao values.')
save('config.json', config)
# The surface pronunciation and reference spelling are deliberately different
# for transliterated terms, as a personal vocabulary recognition task requires.
cases = [
 ('arion-a','mixed','recurring','请打开阿里昂的设置页面。','请打开Arion的设置页面。','Arion'),
 ('arion-b','mixed','recurring','阿里昂这次更新需要重新启动。','Arion这次更新需要重新启动。','Arion'),
 ('mygo-a','mixed','recurring','这个窗口是用麦狗实现的。','这个窗口是用MyGo实现的。','MyGo'),
 ('mygo-b','mixed','recurring','请检查麦狗的按钮样式。','请检查MyGo的按钮样式。','MyGo'),
 ('redis-a','mixed','recurring','缓存服务使用瑞迪斯。','缓存服务使用Redis。','Redis'),
 ('redis-b','mixed','recurring','先别重启瑞迪斯服务。','先别重启Redis服务。','Redis'),
 ('kubernetes-a','mixed','recurring','把服务部署到库伯内蒂斯。','把服务部署到Kubernetes。','Kubernetes'),
 ('kubernetes-b','mixed','recurring','库伯内蒂斯集群暂时不能升级。','Kubernetes集群暂时不能升级。','Kubernetes'),
 ('feishu-a','zh','recurring','把会议记录发到飞书群里。','把会议记录发到飞书群里。','飞书'),
 ('feishu-b','zh','recurring','我没有收到飞书上的消息。','我没有收到飞书上的消息。','飞书'),
 ('safari-a','en','recurring','Open Safari and check the download page.','Open Safari and check the download page.','Safari'),
 ('safari-b','mixed','recurring','请用沙发浏览器打开这个网页。','请用Safari浏览器打开这个网页。','Safari'),
 ('openai-a','en','recurring','We use Open AI models for this project.','We use OpenAI models for this project.','OpenAI'),
 ('openai-b','en','recurring','Please check the Open AI documentation.','Please check the OpenAI documentation.','OpenAI'),
 ('sql-a','en','recurring','Run the sequel query against the database.','Run the SQL query against the database.','SQL'),
 ('sql-b','en','recurring','Do not change the sequel schema.','Do not change the SQL schema.','SQL'),
 ('github-a','mixed','selection_miss','把代码推送到吉特哈布。','把代码推送到GitHub。','GitHub'),
 ('github-b','mixed','selection_miss','请检查吉特哈布上的问题列表。','请检查GitHub上的问题列表。','GitHub'),
 ('codex-a','mixed','selection_miss','用扣代克斯检查这个修改。','用Codex检查这个修改。','Codex'),
 ('codex-b','mixed','selection_miss','扣代克斯已经完成了代码审查。','Codex已经完成了代码审查。','Codex'),
 ('sofa-a','zh','homophone_control','客厅的沙发需要清洗。','客厅的沙发需要清洗。','沙发'),
 ('sofa-b','zh','homophone_control','我想买一个灰色的沙发。','我想买一个灰色的沙发。','沙发'),
 ('sequel-a','en','homophone_control','The sequel to this movie was disappointing.','The sequel to this movie was disappointing.','sequel'),
 ('sequel-b','en','homophone_control','She wrote a sequel to her first novel.','She wrote a sequel to her first novel.','sequel'),
 ('reverse-a','zh','reverse_control','我的眼睛有点不舒服。','我的眼睛有点不舒服。','眼睛'),
 ('reverse-b','zh','reverse_control','燕京大学的历史很有意思。','燕京大学的历史很有意思。','燕京'),
 ('expired-a','mixed','expired_control','向量索引用丘德兰特来管理。','向量索引用Qdrant来管理。','Qdrant'),
 ('expired-b','mixed','expired_control','丘德兰特今天没有报错。','Qdrant今天没有报错。','Qdrant'),
 ('ordinary-zh','zh','ordinary_control','明天下午我们一起去公园散步。','明天下午我们一起去公园散步。',''),
 ('ordinary-en','en','ordinary_control','Please close the window before you leave.','Please close the window before you leave.',''),
]
samples = []
for i,(ident,lang,kind,spoken,reference,term) in enumerate(cases):
    voice = 'Samantha' if lang == 'en' else 'Tingting'
    aiff, wav = out/(ident+'.aiff'), out/(ident+'.wav')
    subprocess.run(['say','-v',voice,'-r','170','-o',str(aiff),spoken],check=True)
    subprocess.run(['ffmpeg','-loglevel','error','-i',str(aiff),'-ar','16000','-ac','1','-c:a','pcm_s16le',str(wav)],check=True)
    with wave.open(str(wav),'rb') as f:
        duration = f.getnframes()/f.getframerate()
    samples.append(dict(id=ident,split='test',sessionId='current-'+ident,
                        speakerId='tts-'+voice,language=lang,unit={'zh':'cer','en':'wer','mixed':'mer'}[lang],
                        kind=kind,spokenText=spoken,reference=reference,term=term,voice=voice,
                        audio=dict(path=wav.name,sha256=hashlib.sha256(wav.read_bytes()).hexdigest(),durationMs=duration*1000)))
# Explicit nonspeech controls with audio bytes, never simulated ASR output.
import numpy as np
rng=np.random.default_rng(1729)
for ident,pcm in [('silence',np.zeros(32000,dtype='<i2')),('noise',(rng.normal(0,150,32000)).astype('<i2'))]:
    wav=out/(ident+'.wav')
    with wave.open(str(wav),'wb') as f:
        f.setnchannels(1);f.setsampwidth(2);f.setframerate(16000);f.writeframes(pcm.tobytes())
    samples.append(dict(id=ident,split='test',sessionId='current-'+ident,speakerId='generated-nonspeech',language='zh',unit='cer',kind='nonspeech',spokenText='',reference='',term='',audio=dict(path=wav.name,sha256=hashlib.sha256(wav.read_bytes()).hexdigest(),durationMs=2000)))
save('corpus.json',dict(version=1,license='AGPL-3.0-only authored fixture text and scripts; audio synthesized locally by macOS system voices for this diagnostic',consent=True,
    provenance='Single developer-authored diagnostic. All memory predates this authored test set. TTS speech, not natural microphone speech. Independent annotations and source-checkpoint parity are absent. Transliterated term references encode approved spelling, not literal Chinese surface rendering.',
    synthesis='macOS say Tingting/Samantha, rate 170; ffmpeg mono PCM16 16kHz; original AIFF retained',samples=samples))
# Warmups use a separate voice and sentence, never test audio or annotations.
subprocess.run(['say','-v','Daniel','-r','170','-o',str(out/'warmup.aiff'),'The meeting room is ready for the next discussion.'],check=True)
subprocess.run(['ffmpeg','-loglevel','error','-i',str(out/'warmup.aiff'),'-ar','16000','-ac','1','-c:a','pcm_s16le',str(out/'warmup.wav')],check=True)
save('warmup.json',dict(split='dev',speakerId='tts-Daniel',sessionId='warmup-only',audio='warmup.wav',sha256=hashlib.sha256((out/'warmup.wav').read_bytes()).hexdigest()))
print('Built',len(samples),'cases')
