"""Reject changed prefix treatments while retaining observed cache/output evidence."""
import copy,unittest
import report
class PrefixEvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows,cls.js=report.checked();cls.m=report.read(report.RUN/'manifest.json');cls.j=report.read(report.RUN/'review.json');cls.d=report.read(report.RUN/'selection.json')
 def corrupt(self,edit):
  rows,m,j,d=map(copy.deepcopy,[self.rows,self.m,self.j,self.d]);edit(rows,m,j,d)
  with self.assertRaises((ValueError,KeyError)):report.validate(rows,m,j,d)
 def test_actual_parity(self):
  result,_,_=report.collect();self.assertEqual((result['normalizedMatchedRequests'],result['identicalRawResponses']),(174,174));self.assertGreater(result['cacheHits'],0);self.assertFalse(result['selection']['qualified'])
 def test_changed_cache_flag(self):self.corrupt(lambda r,m,j,d:r[0]['request'].update(cache_prompt=False))
 def test_impossible_cache_tokens(self):
  def edit(r,m,j,d):r[0]['response']['timings']['cache_n']=10**9;j['reviews'][0]['rowSha256']=report.rowhash(r[0])
  self.corrupt(edit)
 def test_changed_model(self):self.corrupt(lambda r,m,j,d:m['model'].update(sha256='0'*64))
 def test_missing_row(self):self.corrupt(lambda r,m,j,d:r.pop())
 def test_changed_response(self):self.corrupt(lambda r,m,j,d:r[0].update(renderedBody='changed'))
 def test_false_parity_decision(self):self.corrupt(lambda r,m,j,d:d.update(parityQualified=False))
 def test_no_release_promotion(self):self.corrupt(lambda r,m,j,d:d.update(qualified=True))
if __name__=='__main__':unittest.main()
