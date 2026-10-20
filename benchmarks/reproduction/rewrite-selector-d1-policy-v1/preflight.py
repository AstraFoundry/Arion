"""Reuse the frozen typed native executor after native budget qualification."""
import argparse,importlib.util,pathlib,sys,json
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'rewrite-selector-d1-v1'))
import layout
spec=importlib.util.spec_from_file_location('original_d1_native',HERE.parent/'rewrite-selector-d1-v1/preflight.py');native=importlib.util.module_from_spec(spec);spec.loader.exec_module(native)
if __name__=='__main__':
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args()
 qualified=json.loads((args.out.parent/'budget/summary.json').read_text())
 if qualified['status']!='complete' or qualified['modelGenerations']!=0 or qualified['policyRequestsWithoutTruncation']!=6:raise ValueError('Native budget is not qualified')
 native.HERE=HERE;native.validate=layout.validate;sys.argv=[str(__file__),'--out',str(args.out)];native.execute()
