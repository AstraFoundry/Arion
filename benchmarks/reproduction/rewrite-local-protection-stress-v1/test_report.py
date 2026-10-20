"""Check complete stress evidence and raw-meaning review coverage."""
import unittest,sys
import report
sys.path.insert(0,str(report.HERE.parent))
import gguf_evidence_tests
if __name__=='__main__':
 suite=unittest.TestSuite()
 for arm in report.ARMS:suite.addTests(gguf_evidence_tests.suite(report,report.RUN/arm,8))
 result=unittest.TextTestRunner().run(suite);raise SystemExit(not result.wasSuccessful())
