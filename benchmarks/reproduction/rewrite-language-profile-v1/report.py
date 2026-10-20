"""Replay the complete source-bound language recipe study without inference."""
import pathlib,sys,importlib.util
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import gguf_recipe_evidence as evidence
spec=importlib.util.spec_from_file_location('profile_v1_evidence_pipeline',HERE/'pipeline.py');pipeline=importlib.util.module_from_spec(spec);spec.loader.exec_module(pipeline)
RUN=evidence.ROOT/'benchmarks/runs/rewrite-language-profile-v1/run'
read=evidence.read;sha=evidence.sha;rowhash=evidence.rowhash
def checked(directory=RUN):return evidence.checked(HERE,pipeline,directory)
def collect():return evidence.collect(HERE,pipeline,RUN)
if __name__=='__main__':evidence.main(HERE,pipeline,RUN)
