"""Verify user-aligned source evidence, English subspans and visible formatting."""
import json,unittest
import pipeline
C=pipeline.candidates
class PipelineTests(unittest.TestCase):
 def test_letter_normalization_not_homophone_guess(self):
  p=C.prepare('先查 read cash，不改配置。',{'Screen':'readCache、readCash'},'work');h=p['hypotheses'][0];self.assertEqual(h['candidateSpellings'],['readCash']);self.assertEqual(h['sourceLetterMatches'],['readCash']);source,edits=C.apply('{"e0":"replace"}',p);self.assertEqual(source,'先查 readCash，不改配置。')
 def test_different_letters_do_not_get_canonical_priority(self):
  p=C.prepare('先查 write rite，不要重试。',{'Screen':'writeRight、writeRite'},'prompt');self.assertEqual(p['hypotheses'][0]['candidateSpellings'],['writeRite'])
 def test_embedded_identifier_in_english_and_mixed(self):
  for source in ['Please add logging to build cache, but do not change settings.','请 check build cache before deploy，别改设置。']:
   p=C.prepare(source,{'Screen':'build.go: func buildCache(key string)'},'work');self.assertEqual(len(p['hypotheses']),1);h=p['hypotheses'][0];self.assertEqual(h['sourceSpan'],'build cache');self.assertEqual(h['candidateSpellings'],['buildCache']);text,_=C.apply('{"e0":"replace"}',p);self.assertEqual(text,source.replace('build cache','buildCache'))
 def test_quoted_subspan_is_excluded(self):
  self.assertEqual(C.prepare('请原样写“read cash”。',{'Screen':'readCash、readCache'},'prompt')['hypotheses'],[])
 def test_exact_name_and_wrong_pronunciation_excluded(self):
  self.assertEqual(C.prepare('庄雨宁核对，韩若宁您好。',{'Screen':'庄雨宁、庄语宁、韩苦宁'},'email')['hypotheses'],[])
 def test_formatter_uses_complete_visible_source_no_context(self):
  sample=next(x for x in pipeline.read('development.json') if x['id']=='recipient-wrong-relevant');request,p,cleanup=pipeline.selection_payload(sample);edited,edits=C.apply('{"e0":"replace"}',p);request,e=pipeline.format_payload(sample,edited,p,{'e0':'replace'},edits);last=json.loads(request['messages'][-1]['content']);self.assertEqual(last['source'],edited);self.assertIn('林语桐您好',last['source']);self.assertEqual(e['protectedTerms'],[]);self.assertNotIn('Screen',request['messages'][-1]['content'])
 def test_unfiltered_all_outputs(self):
  e=C.protect('read cash',{'hypotheses':[]},{},[])
  for raw in ['','DONE','changed words']:self.assertEqual(C.restore(raw,e),raw)
 def test_all_training_examples_producer_coverage(self):
  for s in pipeline.read('development.json'):pipeline.selection_payload(s)
if __name__=='__main__':unittest.main()
