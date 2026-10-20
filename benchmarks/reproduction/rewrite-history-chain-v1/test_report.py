"""Reject missing historical outputs and altered source/context/outcome evidence."""
import copy,unittest
import report
class HistoricalEvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows,cls.labels=report.checked();cls.m=report.read(report.RUN/'manifest.json');cls.j=report.read(report.RUN/'review.json');cls.o=[report.json.loads(x) for x in (report.RUN/'case-outcomes.jsonl').read_text().splitlines()];cls.d=report.read(report.RUN/'selection.json')
 def corrupt(self,edit):
  r,m,j,o,d=map(copy.deepcopy,[self.rows,self.m,self.j,self.o,self.d]);edit(r,m,j,o,d)
  with self.assertRaises((ValueError,KeyError)):report.validate(r,m,j,o,d)
 def test_complete_scope_and_uncertainty(self):
  s,_,_=report.collect();self.assertEqual((s['sourceCases'],s['newGenerations'],s['actualSelectorCalls']),(401,465,64));self.assertEqual(s['failedFinalIds'],['history-0063','history-0064']);self.assertEqual(s['uncertainFinalIds'],['history-0057','history-0346']);self.assertFalse(s['selection']['qualified'])
 def test_missing_body(self):self.corrupt(lambda r,m,j,o,d:r.pop())
 def test_wrong_context(self):self.corrupt(lambda r,m,j,o,d:r[0].update(context={"Screen":"altered evidence"}))
 def test_hidden_outcome(self):self.corrupt(lambda r,m,j,o,d:o.pop())
 def test_false_release(self):self.corrupt(lambda r,m,j,o,d:d.update(qualified=True))
 def test_unknown_cannot_be_pass(self):
  self.assertTrue(all(not j['sceneUseful'] for j in self.j['reviews'] if j['faithful'] is None))
if __name__=='__main__':unittest.main()
