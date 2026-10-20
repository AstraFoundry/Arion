"""Verify the archived language experiment without inference or private weights."""
import pathlib
import sys
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parent.parent))
from verify_rewrite_archive import verify
if __name__=='__main__':
    verify('rewrite-selector-instruction-language-v1')
