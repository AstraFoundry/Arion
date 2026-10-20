"""Exercise current real observations and immutable prompt/coverage/delivery boundaries."""
import copy,unittest
import report
class SemanticV2EvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows,cls.labels=report.checked();cls.m=report.read(report.RUN/'manifest.json');cls.j=report.read(report.RUN/'review.json');cls.o=list(map(report.json.loads,(report.RUN/'case-outcomes.jsonl').read_text().splitlines()));cls.d=report.read(report.RUN/'selection.json')
 def corrupt(self,edit):
  rows,m,j,o,d=map(copy.deepcopy,[self.rows,self.m,self.j,self.o,self.d]);edit(rows,m,j,o,d);labels={(x['variant'],x['id']):x for x in j['reviews']}
  for r in rows:
   if (r['variant'],r['id']) in labels:labels[(r['variant'],r['id'])]['rowSha256']=report.rowhash(r)
  with self.assertRaises((ValueError,KeyError)):report.validate(rows,m,j,o,d)
 def test_full_actual_scope(self):self.assertEqual((len(self.rows),self.m['formatterGenerations'],self.m['selectorGenerations']),(558,457,101))
 def test_missing_body(self):self.corrupt(lambda r,m,j,o,d:r.pop())
 def test_missing_source_outcome(self):self.corrupt(lambda r,m,j,o,d:o.pop())
 def test_changed_actual_request(self):self.corrupt(lambda r,m,j,o,d:r[0]['request']['messages'][0].update(content='changed policy'))
 def test_wrong_prompt_ownership(self):self.corrupt(lambda r,m,j,o,d:next(x for x in r if x['stage']=='selection').update(templateVersion='old-template'))
 def test_filtered_delivery(self):self.corrupt(lambda r,m,j,o,d:r[0].update(renderedBody='filtered text'))
 def test_unsupported_adoption(self):self.corrupt(lambda r,m,j,o,d:d.update(qualified=True))
 def test_source_fidelity_separate_from_repair(self):
  x=self.labels[('candidate-selection','role-deployment-1')];self.assertTrue(x['faithful']);self.assertFalse(x['choiceConforms']);y=self.labels[('candidate-format','role-deployment-1')];self.assertTrue(y['faithful']);self.assertFalse(y['sceneUseful'])
if __name__=='__main__':unittest.main()
