"""Reject invented native answers, missing reviews and hidden piece truncation."""
import copy,json,pathlib,unittest
from unittest.mock import patch
import report
class NativeEvidenceTests(unittest.TestCase):
 def corrupt_json(self,path,change):
  original=report.read;value=copy.deepcopy(original(path));change(value)
  with patch.object(report,'read',side_effect=lambda p:value if p==path else original(p)):
   with self.assertRaises(ValueError):report.collect()
 def corrupt_text(self,path,change):
  original=pathlib.Path.read_text;value=change(original(path))
  with patch.object(pathlib.Path,'read_text',lambda p,*a,**k:value if p==path else original(p,*a,**k)):
   with self.assertRaises(ValueError):report.collect()
 def test_actual_typed_scope(self):
  summary,items=report.collect();self.assertEqual((len(items),summary['modelResponses'],summary['generatedTokens'],summary['finalBodies']),(72,72,0,0));self.assertEqual(summary['groups']['compact-sentence']['clippedRequests'],12);self.assertFalse(summary['groups']['policy']['qualified'])
 def test_missing_personal_review(self):self.corrupt_json(report.RUN/'review.json',lambda x:x['reviews'].pop())
 def test_false_personal_choice_score(self):self.corrupt_json(report.RUN/'review.json',lambda x:x['reviews'][0].update(choiceConforms=False))
 def test_invented_prose(self):self.corrupt_json(report.RUN/'review.json',lambda x:x['reviews'][0].update(derivedSpellingText='Invented generated body'))
 def test_false_qualification(self):self.corrupt_json(report.RUN/'selection.json',lambda x:x.update(qualified=True))
 def test_hidden_truncation(self):self.corrupt_json(report.ROOT/'benchmarks/runs/rewrite-selector-d1-policy-v1/budget/summary.json',lambda x:x.update(priorTruncatedRequests=0))
 def test_invented_zero_generation_calls(self):self.corrupt_json(report.ROOT/'benchmarks/runs/rewrite-selector-d1-policy-v1/budget/summary.json',lambda x:x.update(modelGenerations=36))
 def test_missing_native_response(self):self.corrupt_text(report.ROOT/'benchmarks/runs/rewrite-selector-d1-v1/preflight/results.jsonl',lambda t:'\n'.join(t.splitlines()[:-1])+'\n')
 def test_changed_complete_ocr(self):
  def change(t):
   rows=list(map(json.loads,t.splitlines()));rows[0]['request']['state']['originalMessages'][-1]['content']='{}';return '\n'.join(json.dumps(r,ensure_ascii=False) for r in rows)+'\n'
  self.corrupt_text(report.ROOT/'benchmarks/runs/rewrite-selector-d1-v1/preflight/results.jsonl',change)
 def test_changed_actual_response_bytes(self):
  path=report.ROOT/'benchmarks/runs/rewrite-selector-d1-v1/preflight/metal/0.response.json';original=pathlib.Path.read_bytes
  with patch.object(pathlib.Path,'read_bytes',lambda p:b'{}' if p==path else original(p)):
   with self.assertRaises(ValueError):report.collect()
 def test_missing_actual_cpu_placement(self):
  path=report.ROOT/'benchmarks/runs/rewrite-selector-d1-v1/preflight/cpu/server.log';original=pathlib.Path.read_bytes
  with patch.object(pathlib.Path,'read_bytes',lambda p:original(p).replace(b'offloaded 0/19 layers to GPU',b'offloaded 19/19 layers to GPU') if p==path else original(p)):
   with self.assertRaises(ValueError):report.collect()
if __name__=='__main__':unittest.main()
