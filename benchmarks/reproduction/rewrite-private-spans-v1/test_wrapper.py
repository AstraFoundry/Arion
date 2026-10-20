"""Test input boundaries, cleanup identity and truly unfiltered restoration."""
import json,unittest
import wrapper
SAMPLES={r['id']:r for r in json.loads((wrapper.HERE/'development.json').read_text())}
class WrapperTests(unittest.TestCase):
 def test_all_sources_and_full_context(self):
  for sample in SAMPLES.values():
   request,encoding=wrapper.payload(sample,'qwen-private-spans');last=request['messages'][-1]['content']
   self.assertTrue(last.endswith(encoding['encodedSource']))
   if sample['context']:self.assertIn(json.dumps(sample['context'],ensure_ascii=False,separators=(',',':')),last)
   self.assertEqual(wrapper.decode(encoding['encodedSource'],encoding),encoding['cleanedSource'])
 def test_literal_private(self):
  request,encoding=wrapper.payload(SAMPLES['lp-stress-literal'],'qwen-private-spans')
  self.assertEqual(encoding['protectedTerms'][0]['kind'],'literal')
  self.assertNotIn('read cash',request['messages'][-1]['content'])
  token=encoding['protectedTerms'][0]['token'];self.assertEqual(wrapper.decode('“'+token+'”',encoding),'“read cash”')
 def test_repeated_written_phrase_uses_same_token(self):
  _,encoding=wrapper.payload(SAMPLES['lp-stress-repeat'],'qwen-private-spans')
  self.assertEqual(len(encoding['protectedTerms']),2);self.assertEqual(len({r['token'] for r in encoding['protectedTerms']}),1)
 def test_no_output_rejection(self):
  _,encoding=wrapper.payload(SAMPLES['lp-stress-condition'],'qwen-private-spans')
  for raw in ['', 'DONE', '<done>', 'unrelated text']:self.assertEqual(wrapper.decode(raw,encoding),raw)
 def test_actual_cleanup_before_encoding(self):
  _,encoding=wrapper.payload(SAMPLES['lp-stress-correction'],'qwen-private-spans')
  self.assertEqual(encoding['cleanedSource'],'先查 load cash，先别改接口。');self.assertNotIn('read cash',encoding['encodedSource'])
 def test_necessary_spelling_evidence_remains_visible(self):
  request,encoding=wrapper.payload(SAMPLES['pe-two-runs'],'qwen-private-spans');last=request['messages'][-1]['content']
  self.assertIn('loadCache',last);self.assertNotIn('sourceSpan":"read cash',last)
if __name__=='__main__':unittest.main()
