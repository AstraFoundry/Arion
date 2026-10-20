"""Test source-relative authority, full-context routing and exact spelling handoff."""
import unittest,json
import pipeline
C=pipeline.candidates
class PipelineTests(unittest.TestCase):
 def test_splice_changes_only_chosen_span(self):
  p=C.prepare('给 fetch cash 加日志，不改配置。',{'Screen':'func fetchCache(key string)'},'work');edited,edits=C.apply('{"e0":"replace"}',p);self.assertEqual(edited,'给 fetchCache 加日志，不改配置。');self.assertEqual(len(edits),1)
 def test_ambiguous_choice_is_bounded(self):
  p=C.prepare('先查 read cash，不改设置。',{'Screen':'readCache、readCash'},'work');self.assertEqual(C.schema(p)['properties']['e0']['enum'],['keep'])
  with self.assertRaises(ValueError):C.apply('{"e0":"replace"}',p)
 def test_literal_and_wrong_pronunciation_not_candidates(self):
  p=C.prepare('韩若宁您好，请原样写“load cash”。',{'Screen':'韩苦宁；loadCache'},'email');self.assertEqual(p['hypotheses'],[])
 def test_already_exact_source_preserved(self):
  p=C.prepare('庄雨宁先核对名单，别提交。',{'Screen':'庄雨宁、庄语宁；名单核对。'},'work');self.assertEqual(p['hypotheses'],[])
 def test_formatter_has_no_screen_text(self):
  sample=next(s for s in pipeline.read('development.json') if s['id']=='p4-work-code');request,p,cleanup=pipeline.selection_payload(sample);self.assertEqual(json.loads(request['messages'][-1]['content'])['context'],sample['context'])
  selection={'e0':'replace'};source,edits=C.apply(json.dumps(selection),p);request,encoding=pipeline.format_payload(sample,source,p,selection,edits);last=json.loads(request['messages'][-1]['content']);self.assertNotIn('context',last);self.assertNotIn('cache.go',request['messages'][-1]['content']);self.assertEqual(C.restore(encoding['encodedSource'],encoding),source)
 def test_unfiltered_body_restoration(self):
  p=C.prepare('请原样写“load cash”。',{'Screen':'loadCache'},'prompt');e=C.protect(p['source'],p,{},[])
  for raw in ['', 'DONE', '<done>']:self.assertEqual(C.restore(raw,e),raw)
 def test_training_examples_match_source_producer(self):
  for s in pipeline.read('development.json'):pipeline.selection_payload(s)
if __name__=='__main__':unittest.main()
