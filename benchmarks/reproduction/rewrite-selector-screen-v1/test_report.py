"""Exercise source coverage, gold separation, actual choices and non-delivery boundaries."""
import copy,unittest
import report
class SelectorScreenEvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows,cls.labels=report.checked();cls.m=report.read(report.RUN/'manifest.json');cls.j=report.read(report.RUN/'review.json');cls.o=list(map(report.json.loads,(report.RUN/'case-outcomes.jsonl').read_text().splitlines()));cls.p=list(map(report.json.loads,(report.RUN/'preparations.jsonl').read_text().splitlines()));cls.d=report.read(report.RUN/'selection.json')
 def corrupt(self,edit):
  rows,m,j,o,p,d=map(copy.deepcopy,[self.rows,self.m,self.j,self.o,self.p,self.d]);edit(rows,m,j,o,p,d);labels={(x['variant'],x['id']):x for x in j['reviews']}
  for r in rows:
   if (r['variant'],r['id']) in labels:labels[(r['variant'],r['id'])]['rowSha256']=report.rowhash(r)
  with self.assertRaises((ValueError,KeyError)):report.validate(rows,m,j,o,p,d)
 def test_actual_scope(self):self.assertEqual((len(self.rows),len(self.p),len(self.o),self.m['formatterGenerations']),(404,457,1828,0))
 def test_missing_skipped_outcome(self):self.corrupt(lambda r,m,j,o,p,d:o.pop(next(i for i,x in enumerate(o) if not x['modelSelectorCalled'])))
 def test_missing_actual_choice(self):self.corrupt(lambda r,m,j,o,p,d:r.pop())
 def test_invented_final_body(self):self.corrupt(lambda r,m,j,o,p,d:r[0].update(renderedBody='fabricated final text'))
 def test_gold_in_actual_request(self):
  def edit(r,m,j,o,p,d):
   x=r[0];data=report.json.loads(x['request']['messages'][-1]['content']);data['allowedChoices']={'e0':['replace']};x['request']['messages'][-1]['content']=report.json.dumps(data,ensure_ascii=False,separators=(',',':'))
  self.corrupt(edit)
 def test_false_conformance_label(self):
  def edit(r,m,j,o,p,d):next(x for x in j['reviews'] if not x['choiceConforms']).update(choiceConforms=True)
  self.corrupt(edit)
 def test_false_screen_promotion(self):self.corrupt(lambda r,m,j,o,p,d:d.update(screenPassedArms=['balanced']))
 def test_optional_same_letter_policy(self):self.assertEqual(report.targets()['tone-control-18']['allowedChoices']['e0'],['keep','replace'])
if __name__=='__main__':unittest.main()
