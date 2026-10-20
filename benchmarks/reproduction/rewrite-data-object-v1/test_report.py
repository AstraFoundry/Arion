"""Test full new evidence and reject changed requests/models/scene/review coverage."""
import sys,unittest
import report
sys.path.insert(0,str(report.HERE.parent))
import gguf_evidence_tests
if __name__=='__main__':
 suite=unittest.TestSuite()
 for arm in report.ARMS:suite.addTests(gguf_evidence_tests.suite(report,report.RUN/arm,36))
 result=unittest.TextTestRunner().run(suite);raise SystemExit(not result.wasSuccessful())
