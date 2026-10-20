"""Verify exact native lexer normalization and retain the aborted orchestration attempt."""
from pathlib import Path
import json,hashlib,shutil
R=Path.cwd();H=R/'benchmarks/reproduction/rewrite-selector-minicpm-v1';W=R/'.workspace/rewrite-selector-minicpm-v1';D=W/'preflight'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def save(p,v):p.write_text(json.dumps(v,ensure_ascii=False,indent=2)+'\n')
assert read(W/'terminal-status.json')['exitCode']==-2
assert not (W/'run/results.jsonl').read_bytes()
A=W/'aborted-before-qualification';A.mkdir(exist_ok=False)
for name in ['run','terminal-status.json','execution.log']:
 shutil.move(str(W/name),str(A/name))
log=(A/'run/selector-model-v1-baseline-work-server.log').read_text()
assert 'Parsed message: {"role":"assistant","content":"{\\"e0\\":\\"keep\\"}"}' in log
save(A/'closure.json',dict(status='aborted',reason='Orchestration incorrectly continued after qualification assertion failed; explicitly interrupted and owned server terminated. One completion is present only in native log; zero captured structured responses. No fabricated response envelope, timing, or quality case.',nativeLogCompletions=1,capturedResponses=0,modelOutputRecoveredFromLog='{"e0":"keep"}',sourceId='history-0095',arm='baseline',probeConforms=False,source='给 load cash 加日志，不记录用户内容，先别改缓存策略。',context={'Screen':'编辑器：cache.go\nfunc loadCache(key string)\n议题：为 loadCache 添加诊断日志'},productionQualified=False))
original=(W/'qualify.py').read_text();shutil.copyfile(W/'qualify.py',H/'original-qualify-failed.py.txt')
replacement="assert read(D/(w['name']+'-props.json'))['chat_template']==read(H/'sources/gguf-header-inspection.json')['metadata']['tokenizer.chat_template'][:-1]"
old="assert read(D/(w['name']+'-props.json'))['chat_template']==read(H/'sources/gguf-header-inspection.json')['metadata']['tokenizer.chat_template']"
assert old in original
script=original.replace(old,replacement).replace("print('Limited runtime qualification passed; all12 semantic task failures retained.')",'')
script=script.replace('All12 actual native completions','Exact pinned native lexer strips one trailing newline; all12 actual native completions')
assert read(H/'sources/gguf-header-inspection.json')['metadata']['tokenizer.chat_template'].endswith('\n')
lexer=(W/'template-diagnostic/common-jinja-lexer.cpp').read_text();assert 'a single trailing newline is stripped if present' in lexer
exec(compile(script,'qualified-native-lexer-check','exec'))
shutil.copytree(W/'template-diagnostic',H/'sources/template-diagnostic')
save(H/'comparison-addendum.json',dict(originalFreezeSha256=sha(H/'freeze.json'),scope='Qualification implementation correction only; no prompt, model, runtime, settings, evaluator, semantic gate or original evidence changed. Initial assertion failed because pinned native lexer removes precisely one trailing newline; this exact transformation, not general whitespace normalization, is now verified. Interrupted accidental attempt retained; future run uses new exclusive directory.',actualTemplatePath='sources/gguf-header-inspection.json',nativeTemplateTransformation='remove exactly one final LF from pinned embedded template',nativeSourceEvidence={str(p.relative_to(H)):sha(p) for p in (H/'sources/template-diagnostic').iterdir() if p.is_file()},plannedNewSelectorResponses=202,abortedNativeLogCompletions=1,capturedAbortedResponses=0))
shutil.copyfile(Path(__file__),H/'original-qualify-v2.py.txt')
print('Native gate corrected from pinned lexer source, not semantic scores; abort retained, ready for exclusive new202 run.')
