"""Verify completed F16 evidence without native workers or weights."""
import pathlib,sys
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parent.parent))
from verify_rewrite_archive import verify
if __name__=='__main__':verify('rewrite-selector-minicpm-f16-v1')
