"""Reject changed bodies, templates and encodings after digest recomputation."""
import unittest,sys
import report
sys.path.insert(0,str(report.HERE.parent))
import gguf_evidence_tests
class ProtectionEvidenceTests(unittest.TestCase):
 def mutate(self,change):
  case=list(gguf_evidence_tests.suite(report,report.RUN/'qwen-protected',28))[0]
  def rebind(rows,review,manifest):
   change(rows,review,manifest);lookup={(r['variant'],r['id']):r for r in rows}
   for judgment in review['reviews']:judgment['rowSha256']=report.evidence.rowhash(lookup[(judgment['variant'],judgment['id'])])
  case.mutate(rebind,True)
 def test_changed_body_rejected(self):
  def change(rows,review,manifest):rows[0]['renderedBody']='Changed body'
  self.mutate(change)
 def test_changed_encoding_rejected(self):
  def change(rows,review,manifest):rows[0]['encoding']['encodedSource']='Changed source'
  self.mutate(change)
 def test_changed_model_rejected(self):
  def change(rows,review,manifest):rows[0]['modelSha256']='0'*64
  self.mutate(change)
 def test_changed_template_rejected(self):
  def change(rows,review,manifest):rows[0]['templateSha256']='0'*64
  self.mutate(change)
if __name__=='__main__':
 suite=unittest.TestSuite()
 for arm in report.runner.ARMS:suite.addTests(gguf_evidence_tests.suite(report,report.RUN/arm,28))
 suite.addTests(unittest.defaultTestLoader.loadTestsFromTestCase(ProtectionEvidenceTests));result=unittest.TextTestRunner().run(suite);raise SystemExit(not result.wasSuccessful())
