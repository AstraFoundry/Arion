"""Check coverage, request reconstruction, metadata binding and serializer restoration."""
import sys,unittest
import report
sys.path.insert(0,str(report.HERE.parent))
import gguf_evidence_tests
class AdapterTests(unittest.TestCase):
 def test_serializer_restored(self):
  original=report.evidence.engine.messages;report.checked(report.RUN/'dev');self.assertIs(report.evidence.engine.messages,original)
 def test_metadata_tamper_after_rehash(self):
  suite=gguf_evidence_tests.suite(report,report.RUN/'dev',48);case=list(suite)[0]
  def change(rows,review,manifest):rows[0]['pronunciationEvidence']=[{'status':'forged'}]
  case.mutate(change,True)
if __name__=='__main__':
 suite=gguf_evidence_tests.suite(report,report.RUN/'dev',48);suite.addTests(unittest.defaultTestLoader.loadTestsFromTestCase(AdapterTests));result=unittest.TextTestRunner().run(suite);raise SystemExit(not result.wasSuccessful())
