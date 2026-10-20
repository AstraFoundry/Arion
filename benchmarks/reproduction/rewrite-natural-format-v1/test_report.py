"""Keep actual source-only inference, uncertainty and source damage distinct."""
import copy,json,unittest
import report
class EvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows=[json.loads(x) for x in (report.RUN/'results.jsonl').read_text().splitlines()];cls.m=report.read(report.RUN/'manifest.json');cls.j=report.read(report.RUN/'review.json');cls.d=report.read(report.RUN/'selection.json')
 def corrupt(self,edit):
  rows=copy.deepcopy(self.rows);j=copy.deepcopy(self.j);edit(rows,j)
  with self.assertRaises((ValueError,KeyError)):report.validate(rows,self.m,j,self.d)
 def test_uncertainty_not_promoted(self):
  result,_,_=report.collect();self.assertEqual(result['summary']['faithful'],109);self.assertEqual(result['summary']['unfaithful'],8);self.assertEqual(result['summary']['uncertain'],3);self.assertEqual(result['summary']['languagePreserved'],116);self.assertFalse(result['productionActivated'])
 def test_unclear_source_not_automatically_usable(self):
  result,_,_=report.collect();self.assertEqual(result['summary']['sourceLimited'],37);self.assertEqual(result['summary']['sceneReady'],72)
 def test_fake_selector(self):self.corrupt(lambda rows,j:rows[0]['sourcePreparation'].update(modelSelectorCalled=True))
 def test_changed_source(self):self.corrupt(lambda rows,j:rows[0].update(source='invented'))
 def test_missing_row(self):self.corrupt(lambda rows,j:rows.pop())
 def test_changed_raw_body(self):self.corrupt(lambda rows,j:rows[0].update(renderedBody='invented'))
 def test_context_injection(self):self.corrupt(lambda rows,j:rows[0].update(actualContextUsed={'Screen':'unexpected'}))
 def test_messages_contain_no_reference_fields(self):
  for r in self.rows:
   data=json.loads(r['request']['messages'][-1]['content']);self.assertEqual(set(data),{'source','opaqueTokens'});self.assertEqual(data['source'],r['editedSource'])
if __name__=='__main__':unittest.main()
