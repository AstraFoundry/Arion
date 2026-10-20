"""Run complete historical/old-control candidate coverage and paired role ablations."""
import pathlib,sys,json
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import gguf_cleanup_chain_v2 as executor
import pipeline
ROOT=executor.engine.ROOT
SOURCES=['benchmarks/corpora/rewrite-history-v1/inputs.json','benchmarks/corpora/rewrite-tone-controls-v1/inputs.json','benchmarks/corpora/rewrite-semantic-roles-v1/inputs.json']
def plans():
 groups=[[dict(sample=s,sourcePath=path) for s in json.loads((ROOT/path).read_text())] for path in SOURCES]
 return [('baseline',groups[2]),('candidate',groups[0]+groups[1]+groups[2])]
if __name__=='__main__':executor.execute(HERE,pipeline,plans(),dict(pipeline='selector-semantic-v1',sourceCases=457,plannedSourceAttempts=489,newAsrGenerations=0,newOcrGenerations=0,scope=json.loads((HERE/'protocol.json').read_text())['limits']))
