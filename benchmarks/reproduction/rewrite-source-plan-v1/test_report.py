"""Check complete plans and reject changed schema/body after hash recomputation."""
import unittest,sys
import report
sys.path.insert(0,str(report.HERE.parent))
import gguf_evidence_tests
class PlanEvidenceTests(unittest.TestCase):
 def mutate(self,change):
  suite=gguf_evidence_tests.suite(report,report.RUN/'dev',56);case=list(suite)[0]
  def rebind(rows,review,manifest):
   change(rows,review,manifest);lookup={(r['variant'],r['id']):r for r in rows}
   for judgment in review['reviews']:judgment['rowSha256']=report.evidence.rowhash(lookup[(judgment['variant'],judgment['id'])])
  case.mutate(rebind,True)
 def test_changed_constructed_body(self):
  def change(rows,review,manifest):rows[0]['renderedBody']='Fabricated body'
  self.mutate(change)
 def test_changed_prepared_units(self):
  def change(rows,review,manifest):
   row=next(r for r in rows if r['variant']=='plan');row['preparedPlan']['units'][0]['source']='Altered unit'
  self.mutate(change)
 def test_changed_schema(self):
  def change(rows,review,manifest):rows[0]['request']['response_format']={'type':'json_object'}
  self.mutate(change)
if __name__=='__main__':
 suite=gguf_evidence_tests.suite(report,report.RUN/'dev',56);suite.addTests(unittest.defaultTestLoader.loadTestsFromTestCase(PlanEvidenceTests));result=unittest.TextTestRunner().run(suite);raise SystemExit(not result.wasSuccessful())
