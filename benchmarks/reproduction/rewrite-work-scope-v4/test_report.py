"""Validate work formatter evidence with meaningful corruption probes."""
import pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import report
from gguf_work_scope_tests import run
if __name__=='__main__':run(report.study,110)
