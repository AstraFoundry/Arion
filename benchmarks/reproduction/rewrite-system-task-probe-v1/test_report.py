"""Test exact requests, model identity, review coverage and scene integrity."""
import unittest,sys
import report
sys.path.insert(0,str(report.HERE.parent))
import gguf_evidence_tests
if __name__=='__main__':
 suite=unittest.TestSuite()
 for arm in ['dev']:suite.addTests(gguf_evidence_tests.suite(report,report.RUN/arm,12))
 result=unittest.TextTestRunner().run(suite);raise SystemExit(not result.wasSuccessful())
