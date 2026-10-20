"""Execute the full457-source chain with candidate sentence previews."""
import pathlib,sys,json
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import gguf_cleanup_chain_v3 as executor
import pipeline
ROOT=executor.engine.ROOT
SOURCES=['benchmarks/corpora/rewrite-history-v1/inputs.json','benchmarks/corpora/rewrite-tone-controls-v1/inputs.json','benchmarks/corpora/rewrite-semantic-roles-v1/inputs.json']
def plans():return [('candidate',[dict(sample=s,sourcePath=p) for p in SOURCES for s in json.loads((ROOT/p).read_text())])]
if __name__=='__main__':executor.execute(HERE,pipeline,plans(),dict(pipeline='selector-preview-v1',sourceCases=457,plannedSourceAttempts=457,newAsrGenerations=0,newOcrGenerations=0,scope=json.loads((HERE/'protocol.json').read_text())['limits']))
