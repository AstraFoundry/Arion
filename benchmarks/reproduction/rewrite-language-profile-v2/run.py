"""Execute the frozen source-script policy/example routing recipe."""
import pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import gguf_paired_formatter
import pipeline
if __name__=='__main__':gguf_paired_formatter.execute(HERE,pipeline)
