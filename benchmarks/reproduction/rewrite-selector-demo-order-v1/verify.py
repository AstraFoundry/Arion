"""Verify archived research evidence without weights or native workers."""
import pathlib,sys
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parent.parent))
from verify_rewrite_archive import verify
if __name__=='__main__':verify('rewrite-selector-demo-order-v1')
