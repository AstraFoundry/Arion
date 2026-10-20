"""Reuse the immutable native typed executor for each declared layout."""
import argparse,importlib.util,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'rewrite-selector-d1-v1'))
import layout
spec=importlib.util.spec_from_file_location('original_d1_native',HERE.parent/'rewrite-selector-d1-v1/preflight.py');native=importlib.util.module_from_spec(spec);spec.loader.exec_module(native)
if __name__=='__main__':
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();args.out.mkdir(parents=True,exist_ok=False)
 for kind in ['structured','sentence']:
  native.HERE=HERE/kind;native.validate=lambda records,k=kind:layout.validate(records,k)
  sys.argv=[str(__file__),'--out',str(args.out/kind)];native.execute()
