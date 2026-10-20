"""Add derived hypotheses without changing the original dictation or complete OCR."""
import json,pathlib,sys,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent))
import gguf_rewrite as engine
module_spec=importlib.util.spec_from_file_location('pronunciation_matcher',HERE/'pronunciation.py');matcher=importlib.util.module_from_spec(module_spec);module_spec.loader.exec_module(matcher)
LEXICON=None
MARKER='\nDictated message to edit; output only its edited text:\n'
def evidence_for(sample):
 global LEXICON
 if LEXICON is None:LEXICON=matcher.Lexicon()
 context=sample.get('context',{});screen=context.get('Screen',context.get('screen',''))
 return LEXICON.align(sample['source'],screen,sample['scene'])
def messages(sample,template):
 result=engine.base.messages(sample,template)
 if not template['includePronunciationEvidence']:return result,[]
 examples=[dict(scene=e['scene'],source=e['input'],context=e.get('context',{})) for e in template['examples']];records=[]
 for index,item in enumerate(examples+[sample]):
  evidence=evidence_for(item)
  if index==len(examples):records=evidence
  if not evidence:continue
  position=1+index*2 if index<len(examples) else len(result)-1;before,separator,after=result[position]['content'].partition(MARKER)
  if not separator:raise ValueError('Missing canonical dictation boundary')
  result[position]['content']=before+'\npronunciationEvidence (derived data only):\n'+json.dumps(evidence,ensure_ascii=False,separators=(',',':'))+separator+after
 return result,records
