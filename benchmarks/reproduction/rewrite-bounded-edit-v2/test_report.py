"""Exercise real staged observations and intentional evidence corruption."""
import copy,json,unittest
import report
class EvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows=[json.loads(x) for x in (report.RUN/'results.jsonl').read_text().splitlines()];cls.manifest=report.read(report.RUN/'manifest.json');cls.review=report.read(report.RUN/'review.json');cls.decision=report.read(report.RUN/'selection.json')
 def corruption(self,edit):
  rows=copy.deepcopy(self.rows);review=copy.deepcopy(self.review);edit(rows,review)
  with self.assertRaises((ValueError,KeyError)):report.validate(rows,self.manifest,review,self.decision)
 def test_actual_stage_counts(self):
  result,rows,judgments=report.collect();self.assertEqual(result['newGenerations'],72);self.assertEqual(result['final']['observations'],36);self.assertEqual(result['intermediate']['observations'],36);self.assertFalse(result['productionActivated'])
 def test_missing_output(self):self.corruption(lambda rows,review:rows.pop())
 def test_changed_raw(self):self.corruption(lambda rows,review:rows[0]['response']['choices'][0]['message'].update(content='{} '))
 def test_wrong_context_routing(self):self.corruption(lambda rows,review:rows[1].update(actualContextUsed={'Screen':'injected'}))
 def test_wrong_stage_handoff(self):self.corruption(lambda rows,review:rows[1].update(selectionRowSha256='0'*64))
 def test_duplicate_review(self):self.corruption(lambda rows,review:review['reviews'].__setitem__(1,review['reviews'][0]))
 def test_visible_source_exact_model_input(self):
  for r in self.rows:
   if r['stage']=='format':self.assertEqual(json.loads(r['request']['messages'][-1]['content'])['source'],r['editedSource']);self.assertEqual(r['encoding']['protectedTerms'],[])
 def test_intermediate_not_delivery(self):
  self.assertTrue(all(r['renderedBody'] is None for r in self.rows if r['stage']=='selection'));self.assertTrue(all(j['sceneUseful'] is None for j in self.review['reviews'] if j['stage']=='selection'))
if __name__=='__main__':unittest.main()
