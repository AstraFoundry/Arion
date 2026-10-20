"""Ensure caching never changes source, schema, prompt or generation controls."""
import unittest,json
import pipeline
class PrefixPayloadTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.inputs=json.loads((pipeline.HERE/'inputs.json').read_text());cls.cleanups={r['id']:r['cleanup'] for r in json.loads((pipeline.HERE.parent.parent/'runs/rewrite-cleanup-copula-v1/observations.json').read_text())}
 def test_selector_only_cache_changes(self):
  for s in self.inputs:
   a,p,c=pipeline.selection_payload(s,self.cleanups[s['id']]);b,q,d=pipeline.prior.selection_payload(s,self.cleanups[s['id']]);self.assertTrue(a.pop('cache_prompt'));self.assertFalse(b.pop('cache_prompt'));self.assertEqual((a,p,c),(b,q,d))
 def test_formatter_only_cache_changes(self):
  for s in self.inputs:
   _,p,c=pipeline.selection_payload(s,self.cleanups[s['id']]);a,e=pipeline.format_payload(s,c['text'],p,{},[]);b,f=pipeline.prior.format_payload(s,c['text'],p,{},[]);self.assertTrue(a.pop('cache_prompt'));self.assertFalse(b.pop('cache_prompt'));self.assertEqual((a,e),(b,f))
if __name__=='__main__':unittest.main()
