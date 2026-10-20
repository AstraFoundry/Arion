"""Verify full36 native observations and reject changed evidence identities."""
import sys,unittest
import report
sys.path.insert(0,str(report.HERE.parent))
import gguf_evidence_tests
if __name__=='__main__':
 result=unittest.TextTestRunner().run(gguf_evidence_tests.suite(report,report.RUN/'qwen-q8-private',36));raise SystemExit(not result.wasSuccessful())
