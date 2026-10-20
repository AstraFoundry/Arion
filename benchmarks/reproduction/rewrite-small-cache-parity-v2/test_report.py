"""Check full coverage and tamper rejection for this frozen diagnostic."""
import sys,pathlib,unittest
import report
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parent.parent))
import gguf_evidence_tests
if __name__=="__main__":
    result=unittest.TextTestRunner().run(gguf_evidence_tests.suite(report,report.RUN/"run",40))
    raise SystemExit(not result.wasSuccessful())
