"""Reject broken source identity, fabricated skipped calls and timing claims."""
import copy,unittest
import report
class EvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows,cls.js=report.checked();cls.m=report.read(report.RUN/'manifest.json');cls.j=report.read(report.RUN/'review.json');cls.d=report.read(report.RUN/'selection.json')
 def corrupt(self,edit):
  rows,m,j,d=map(copy.deepcopy,[self.rows,self.m,self.j,self.d]);edit(rows,m,j,d)
  with self.assertRaises((ValueError,KeyError,TypeError)):report.validate(rows,m,j,d)
 def test_actual_call_coverage(self):
  result,_,_=report.collect();self.assertEqual((result['newGenerations'],result['actualSelectorCalls'],result['finalBodies']),(174,18,156));self.assertFalse(result['selection']['qualified'])
 def test_wrong_cleanup(self):self.corrupt(lambda r,m,j,d:r[0]['cleanup'].update(version='speech-spans-v1'))
 def test_wrong_source(self):self.corrupt(lambda r,m,j,d:r[0].update(source='changed'))
 def test_fake_skipped_call(self):
  def edit(rows,m,j,d):r=next(r for r in rows if r['stage']=='format' and not r['sourcePreparation']['modelSelectorCalled']);r['sourcePreparation']['modelSelectorCalled']=True
  self.corrupt(edit)
 def test_fake_selection_reference(self):
  def edit(rows,m,j,d):r=next(r for r in rows if r['stage']=='format' and r['sourcePreparation']['modelSelectorCalled']);r['selectionRowSha256']='0'*64
  self.corrupt(edit)
 def test_changed_full_context(self):self.corrupt(lambda r,m,j,d:next(r for r in r if r['stage']=='selection')['actualContextUsed'].clear())
 def test_invalid_continuous_timer(self):
  def edit(rows,m,j,d):r=next(r for r in rows if r['stage']=='format');r['pipelineMs']=1;j['reviews'][0]['rowSha256']=report.rowhash(r)
  self.corrupt(edit)
 def test_unknown_not_usable(self):
  def edit(rows,m,j,d):next(x for x in j['reviews'] if x['faithful'] is None)['sceneUseful']=True
  self.corrupt(edit)
 def test_release_not_promoted(self):self.corrupt(lambda r,m,j,d:d.update(qualified=True))
if __name__=='__main__':unittest.main()
