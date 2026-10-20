"""Detect altered source-bound payloads and uncertain-meaning promotion."""
import copy,unittest
import report
class EvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows,cls.labels=report.checked();cls.m=report.read(report.RUN/'manifest.json');cls.j=report.read(report.RUN/'review.json');cls.d=report.read(report.RUN/'selection.json')
 def validate(self,rows,m,j,d):return report.evidence.validate(report.HERE,report.pipeline,rows,m,j,d,report.RUN)
 def corrupt(self,edit):
  rows,m,j,d=map(copy.deepcopy,[self.rows,self.m,self.j,self.d]);edit(rows,m,j,d)
  with self.assertRaises((ValueError,KeyError)):self.validate(rows,m,j,d)
 def test_full_evidence(self):
  result,_,_=report.collect();n=result['groups']['natural'];self.assertEqual((n['faithful'],n['sceneReady'],n['uncertain']),(119,75,1));self.assertFalse(result['selection']['qualified'])
 def test_changed_source(self):self.corrupt(lambda r,m,j,d:r[0].update(source='changed'))
 def test_changed_payload(self):self.corrupt(lambda r,m,j,d:r[0]['request'].update(max_tokens=10))
 def test_changed_placement(self):
  def edit(r,m,j,d):c=m['workers'][0]['command'];c[c.index('--n-gpu-layers')+1]='0'
  self.corrupt(edit)
 def test_missing_row(self):self.corrupt(lambda r,m,j,d:r.pop())
 def test_unknown_not_usable(self):
  def edit(r,m,j,d):next(x for x in j['reviews'] if x['faithful'] is None)['sceneUseful']=True
  self.corrupt(edit)
 def test_no_release_promotion(self):self.corrupt(lambda r,m,j,d:d.update(qualified=True))
if __name__=='__main__':unittest.main()
