"""Verify sentence-choice evidence without loading native inference."""
import pathlib
import sys
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent))
from verify_rewrite_archive import verify
if __name__ == '__main__':
    verify('rewrite-selector-sentence-choice-v1')
