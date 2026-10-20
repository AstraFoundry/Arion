"""Reject incomplete full-chain coverage and altered inputs/edits/delivery/release claims."""
import copy,unittest
import report
class ToneChainEvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows,cls.labels=report.checked();cls.m=report.read(report.RUN/'manifest.json');cls.j=report.read(report.RUN/'review.json');cls.o=list(map(report.json.loads,(report.RUN/'case-outcomes.jsonl').read_text().splitlines()));cls.d=report.read(report.RUN/'selection.json')
 def corrupt(self,edit):
  rows,m,j,o,d=map(copy.deepcopy,[self.rows,self.m,self.j,self.o,self.d]);edit(rows,m,j,o,d);labels={(x['variant'],x['id']):x for x in j['reviews']}
  for r in rows:
   if (r['variant'],r['id']) in labels:labels[(r['variant'],r['id'])]['rowSha256']=report.rowhash(r)
  with self.assertRaises((ValueError,KeyError)):report.validate(rows,m,j,o,d)
 def test_complete_actual_counts(self):self.assertEqual((len(self.rows),self.m['formatterGenerations'],self.m['selectorGenerations']),(538,449,89))
 def test_missing_final_body(self):self.corrupt(lambda r,m,j,o,d:r.pop())
 def test_wrong_candidate_context(self):self.corrupt(lambda r,m,j,o,d:r[0].update(context={'Screen':'unobserved evidence'}))
 def test_filtered_body(self):self.corrupt(lambda r,m,j,o,d:r[0].update(renderedBody='replacement text'))
 def test_missing_source_outcome(self):self.corrupt(lambda r,m,j,o,d:o.pop())
 def test_false_release(self):self.corrupt(lambda r,m,j,o,d:d.update(qualified=True))
 def test_same_letter_control_is_allowed(self):
  js=[x for x in self.j['reviews'] if x['stage']=='selection' and x['id']=='tone-control-18'];self.assertEqual(len(js),2);self.assertTrue(all(x['choiceConforms'] for x in js))
if __name__=='__main__':unittest.main()
