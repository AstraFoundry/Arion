"""Reject omitted comparisons, new spelling decisions and invented full-chain timing."""
import copy,unittest
import report
class FormatterIsolationTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows,cls.js=report.checked();cls.m=report.read(report.RUN/'manifest.json');cls.j=report.read(report.RUN/'review.json');cls.d=report.read(report.RUN/'selection.json')
 def corrupt(self,edit):
  r,m,j,d=map(copy.deepcopy,[self.rows,self.m,self.j,self.d]);edit(r,m,j,d)
  labels={(x['variant'],x['id']):x for x in j['reviews']}
  for row in r:
   if (row['variant'],row['id']) in labels:labels[(row['variant'],row['id'])]['rowSha256']=report.rowhash(row)
  with self.assertRaises((ValueError,KeyError)):report.validate(r,m,j,d)
 def test_all_sources_attempted(self):self.assertEqual(len(self.rows),401)
 def test_missing_case(self):self.corrupt(lambda r,m,j,d:r.pop())
 def test_changed_prior_spelling(self):self.corrupt(lambda r,m,j,d:r[0].update(editedSource='unsupported replacement'))
 def test_invented_pipeline_latency(self):self.corrupt(lambda r,m,j,d:r[0].update(pipelineMs=1))
 def test_unreviewed_output(self):self.corrupt(lambda r,m,j,d:j['reviews'].pop())
 def test_false_release(self):self.corrupt(lambda r,m,j,d:d.update(qualified=True))
if __name__=='__main__':unittest.main()
