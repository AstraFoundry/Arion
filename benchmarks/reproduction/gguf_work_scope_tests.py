"""Corrupt source/request/cleanup/delivery evidence while preserving review row hashes."""
import copy,unittest
from gguf_work_scope_report import read,rowhash
def run(study,expected_count):
 class WorkEvidenceTests(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
   cls.rows,cls.labels=study.checked();cls.m=read(study.run/'manifest.json');cls.j=read(study.run/'review.json');cls.d=read(study.run/'selection.json')
  def corrupt(self,edit):
   rows,m,j,d=map(copy.deepcopy,[self.rows,self.m,self.j,self.d]);edit(rows,m,j,d);labels={(x['variant'],x['id']):x for x in j['reviews']}
   for r in rows:
    if (r['variant'],r['id']) in labels:labels[(r['variant'],r['id'])]['rowSha256']=rowhash(r)
   with self.assertRaises((ValueError,KeyError)):study.validate(rows,m,j,d)
  def test_complete_scope(self):self.assertEqual(len(self.rows),expected_count)
  def test_missing_body(self):self.corrupt(lambda r,m,j,d:r.pop())
  def test_injected_model_context(self):self.corrupt(lambda r,m,j,d:r[0]['request']['messages'][-1].update(content='unsupported OCR facts'))
  def test_invented_control_cleanup(self):self.corrupt(lambda r,m,j,d:next(x for x in r if x['cohort']=='new-synthetic-work-controls')['cleanup'].update(version='speech-spans-v2'))
  def test_filtered_delivery(self):self.corrupt(lambda r,m,j,d:r[0].update(renderedBody='replacement body'))
  def test_false_release(self):self.corrupt(lambda r,m,j,d:d.update(qualified=True))
 suite=unittest.defaultTestLoader.loadTestsFromTestCase(WorkEvidenceTests);result=unittest.TextTestRunner(verbosity=1).run(suite)
 if not result.wasSuccessful():raise SystemExit(1)
