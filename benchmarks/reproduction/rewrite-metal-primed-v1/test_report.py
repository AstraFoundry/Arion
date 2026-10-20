"""Reject omitted warmups, invented readiness and changed canary/source evidence."""
import copy,unittest
import report
class PrimingEvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.d=report.BASE/'forward';cls.rows,cls.j=report.checked(cls.d);cls.warm=[r for r in cls.rows if r.get('role')=='warmup'];cls.wj=report.read(cls.d/'warmup-review.json');cls.s=report.read(cls.d/'study.json');cls.m=report.read(cls.d/'manifest.json')
 def corrupt(self,edit):
  rows,j,s,m=map(copy.deepcopy,[self.warm,self.wj,self.s,self.m]);edit(rows,j,s,m)
  with self.assertRaises((ValueError,KeyError)):report.validate_warmups(rows,m,j,s,m,self.d)
 def test_complete_measured_and_priming_scope(self):
  result,rows,_=report.collect();self.assertEqual((result['newGenerations'],len(rows),result['warmupGenerations']),(380,380,32));self.assertEqual(set(result['groups']),{'forward','reverse'})
 def test_missing_warmup(self):self.corrupt(lambda r,j,s,m:r.pop())
 def test_changed_marker(self):self.corrupt(lambda r,j,s,m:r[1].update(canary='changed'))
 def test_hidden_readiness_cost(self):self.corrupt(lambda r,j,s,m:s['workers'][0].update(primingMs=0))
 def test_false_zero_generation(self):self.corrupt(lambda r,j,s,m:s.update(warmupGenerations=0))
 def test_changed_actual_cleanup(self):self.corrupt(lambda r,j,s,m:r[0]['cleanup'].update(version='invented'))
if __name__=='__main__':unittest.main()
