"""Reject altered staged evidence, context routing and intermediate delivery labels."""
import copy,json,unittest
import report
class EvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows=[json.loads(x) for x in (report.RUN/'results.jsonl').read_text().splitlines()];cls.manifest=report.read(report.RUN/'manifest.json');cls.review=report.read(report.RUN/'review.json');cls.decision=report.read(report.RUN/'selection.json')
 def check_corruption(self,change):
  rows=copy.deepcopy(self.rows);review=copy.deepcopy(self.review);change(rows,review)
  with self.assertRaises((ValueError,KeyError)):report.validate(rows,self.manifest,review,self.decision)
 def test_complete_separate_stage_counts(self):
  result,_,_=report.collect();self.assertEqual(result['intermediate']['faithful'],33);self.assertEqual(result['final']['faithful'],28);self.assertEqual(result['final']['repairs'],6);self.assertFalse(result['productionActivated'])
 def test_missing_output_rejected(self):self.check_corruption(lambda rows,review:rows.pop())
 def test_changed_raw_output_rejected(self):self.check_corruption(lambda rows,review:rows[0]['response']['choices'][0]['message'].update(content='{} '))
 def test_injected_formatter_context_rejected(self):self.check_corruption(lambda rows,review:rows[1].update(actualContextUsed={'Screen':'unrelated'}))
 def test_changed_handoff_rejected(self):self.check_corruption(lambda rows,review:rows[1].update(selectionRowSha256='0'*64))
 def test_duplicate_review_rejected(self):self.check_corruption(lambda rows,review:review['reviews'].__setitem__(1,review['reviews'][0]))
 def test_selection_not_delivery(self):
  self.assertTrue(all(r['renderedBody'] is None for r in self.rows if r['stage']=='selection'));self.assertTrue(all(r['sceneUseful'] is None for r in self.review['reviews'] if r['stage']=='selection'))
if __name__=='__main__':unittest.main()
