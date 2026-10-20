"""Run shared archive-integrity tests without model execution."""
import sys,unittest
import report
sys.path.insert(0,str(report.HERE.parent))
from review_evidence_tests import suite
if __name__=='__main__':
    result=unittest.TextTestRunner().run(suite(report,report.RUN/'dev',24))
    raise SystemExit(0 if result.wasSuccessful() else 1)
