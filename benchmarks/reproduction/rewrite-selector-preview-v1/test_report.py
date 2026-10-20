"""Reject altered actual previews, delivery, source scope and development claims."""
import copy,unittest
import report
class PreviewEvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows,cls.labels=report.checked();cls.m=report.read(report.RUN/'manifest.json');cls.j=report.read(report.RUN/'review.json');cls.o=list(map(report.json.loads,(report.RUN/'case-outcomes.jsonl').read_text().splitlines()));cls.d=report.read(report.RUN/'selection.json')
 def corrupt(self,edit):
  rows,m,j,o,d=map(copy.deepcopy,[self.rows,self.m,self.j,self.o,self.d]);edit(rows,m,j,o,d);labels={(x['variant'],x['id']):x for x in j['reviews']}
  for r in rows:
   if (r['variant'],r['id']) in labels:labels[(r['variant'],r['id'])]['rowSha256']=report.rowhash(r)
  with self.assertRaises((ValueError,KeyError)):report.validate(rows,m,j,o,d)
 def test_complete_actual_scope(self):self.assertEqual((len(self.rows),self.m['formatterGenerations'],self.m['selectorGenerations']),(558,457,101))
 def test_missing_body(self):self.corrupt(lambda r,m,j,o,d:r.pop())
 def test_missing_outcome(self):self.corrupt(lambda r,m,j,o,d:o.pop())
 def test_changed_preview(self):
  def change(r,m,j,o,d):
   x=next(x for x in r if x['stage']=='selection');data=report.json.loads(x['request']['messages'][-1]['content']);data['hypotheses'][0]['candidateRewrites'][0]['sourceAfterReplacement']='unobserved text';x['request']['messages'][-1]['content']=report.json.dumps(data,ensure_ascii=False,separators=(',',':'))
  self.corrupt(change)
 def test_changed_template_owner(self):self.corrupt(lambda r,m,j,o,d:r[0].update(templatePath='wrong-prompts.json'))
 def test_filtered_delivery(self):self.corrupt(lambda r,m,j,o,d:r[0].update(renderedBody='filtered'))
 def test_false_development_claim(self):self.corrupt(lambda r,m,j,o,d:d.update(developmentPassed=True))
 def test_negative_target_is_not_a_success(self):
  j=self.labels[('candidate-format','role-nursery-1')];self.assertFalse(j['faithful']);self.assertFalse(j['sceneUseful']);self.assertFalse(j['editSupported'])
if __name__=='__main__':unittest.main()
