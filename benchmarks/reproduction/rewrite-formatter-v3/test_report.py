"""Reject altered formatter evidence and fabricated intermediate reuse."""
import copy,json,unittest
import report
class EvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows=[json.loads(x) for x in (report.RUN/'results.jsonl').read_text().splitlines()];cls.m=report.read(report.RUN/'manifest.json');cls.j=report.read(report.RUN/'review.json');cls.d=report.read(report.RUN/'selection.json')
 def corrupt(self,edit):
  rows=copy.deepcopy(self.rows);review=copy.deepcopy(self.j);edit(rows,review)
  with self.assertRaises((ValueError,KeyError)):report.validate(rows,self.m,review,self.d)
 def test_actual_counts_not_double_counted(self):
  result,_,_=report.collect();self.assertEqual(result['newGenerations'],36);self.assertEqual(result['reusedActualSelections'],36);self.assertEqual(result['identicalEditedSources'],36);self.assertEqual(result['final']['faithful'],35);self.assertFalse(result['productionActivated'])
 def test_changed_reuse_identity(self):self.corrupt(lambda rows,j:rows[0].update(selectionRowSha256='0'*64))
 def test_changed_raw_body(self):self.corrupt(lambda rows,j:rows[0].update(renderedBody='invented'))
 def test_injected_context(self):self.corrupt(lambda rows,j:rows[0].update(actualContextUsed={'Screen':'unexpected'}))
 def test_missing_observation(self):self.corrupt(lambda rows,j:rows.pop())
 def test_duplicate_review(self):self.corrupt(lambda rows,j:j['reviews'].__setitem__(1,j['reviews'][0]))
 def test_four_seconds_not_misreported_as_wrong_value(self):
  row=next(r for r in self.rows if r['id']=='f-prompt-size');j=next(j for j in self.j['reviews'] if j['id']==row['id']);self.assertIn('四秒',row['renderedBody']);self.assertFalse(j['faithful']);self.assertIn('数值',j['reviewCaveat'])
 def test_no_fabricated_selector_observations(self):self.assertTrue(all(r['stage']=='format' for r in self.rows))
if __name__=='__main__':unittest.main()
