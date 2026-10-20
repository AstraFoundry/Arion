"""Verify stage isolation and uniform absence-of-hypotheses behavior."""
import unittest,json,pathlib
import pipeline
class PayloadTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.inputs=json.loads((pipeline.HERE/'inputs.json').read_text());cls.cleanups={r['id']:r['cleanup'] for r in json.loads((pipeline.HERE.parent.parent/'runs/rewrite-cleanup-copula-v1/observations.json').read_text())}
 def test_candidate_call_coverage(self):
  prepared=[pipeline.selection_payload(s,self.cleanups[s['id']])[1] for s in self.inputs]
  self.assertEqual(sum(bool(p['hypotheses']) for p in prepared),18)
 def test_actual_cleanup_used(self):
  sample=next(s for s in self.inputs if s['id']=='sp-self-correct');_,p,c=pipeline.selection_payload(sample,self.cleanups[sample['id']]);self.assertEqual(p['source'],'把超时设成四秒，保留并发数，先别改其他配置。');self.assertEqual(c['version'],'speech-spans-v2')
 def test_no_context_in_format(self):
  s=next(s for s in self.inputs if s['id']=='lp-stress-injection');_,p,c=pipeline.selection_payload(s,self.cleanups[s['id']]);r,_=pipeline.format_payload(s,c['text'],p,{},[]);self.assertNotIn(s['context']['Screen'],str(r))
 def test_natural_no_selector_candidates(self):
  development={s['id'] for s in json.loads((pipeline.HERE.parent/'rewrite-bounded-edit-v2/development.json').read_text())}
  for s in self.inputs:
   if s['id'] not in development:self.assertFalse(pipeline.selection_payload(s,self.cleanups[s['id']])[1]['hypotheses'])
if __name__=='__main__':unittest.main()
