"""Reject altered provider treatments and raw-source controls."""
import copy,json,unittest
import report
class EvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows=[json.loads(x) for x in (report.RUN/'results.jsonl').read_text().splitlines()];cls.m=report.read(report.RUN/'manifest.json');cls.j=report.read(report.RUN/'review.json');cls.d=report.read(report.RUN/'selection.json')
 def corrupt(self,edit):
  rows=copy.deepcopy(self.rows);j=copy.deepcopy(self.j);m=copy.deepcopy(self.m);edit(rows,j,m)
  with self.assertRaises((ValueError,KeyError)):report.validate(rows,m,j,self.d)
 def test_all_pairs_and_differences(self):
  result,_,_=report.collect();self.assertEqual(result['identicalRequests'],36);self.assertEqual(result['identicalBodies'],34);self.assertEqual(result['faithful'],35);self.assertEqual(result['repairs'],11)
 def test_wrong_provider(self):
  def edit(rows,j,m):c=m['workers'][0]['command'];c[c.index('--n-gpu-layers')+1]='0'
  self.corrupt(edit)
 def test_wrong_pair(self):self.corrupt(lambda rows,j,m:rows[0]['sourceControl'].update(rowSha256='0'*64))
 def test_wrong_request(self):self.corrupt(lambda rows,j,m:rows[0]['request'].update(max_tokens=12))
 def test_missing_row(self):self.corrupt(lambda rows,j,m:rows.pop())
 def test_changed_body(self):self.corrupt(lambda rows,j,m:rows[0].update(renderedBody='changed'))
if __name__=='__main__':unittest.main()
