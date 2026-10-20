"""Reuse frozen work job construction and regenerate every candidate work source."""
import pathlib,sys,importlib.util,json
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import gguf_formatter_jobs as executor
import pipeline
s=importlib.util.spec_from_file_location('work_v4_job_source',HERE.parent/'rewrite-work-scope-v3/run.py');prior=importlib.util.module_from_spec(s);s.loader.exec_module(prior);prior.HERE=HERE;prior.pipeline=pipeline
def jobs():
 return [dict(j,pipeline='work-scope-v4',templateVersion='work-scope-v4-work') for j in prior.jobs() if j['variant']=='candidate']
if __name__=='__main__':executor.execute(HERE,jobs(),dict(pipeline='work-scope-v4',sourceCases=110,historicalCases=94,newControls=16,newSelectorGenerations=0,newAsrGenerations=0,scope=json.loads((HERE/'protocol.json').read_text())['limits']))
