"""Run all401 historical sources plus paired24 authored OCR controls."""
import pathlib,sys,json
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import gguf_cleanup_chain_v2 as executor
import pipeline
ROOT=executor.engine.ROOT
HISTORY='benchmarks/corpora/rewrite-history-v1/inputs.json';CONTROLS='benchmarks/corpora/rewrite-tone-controls-v1/inputs.json'
def plans():
 historical=[dict(sample=s,sourcePath=HISTORY) for s in json.loads((ROOT/HISTORY).read_text())];controls=[dict(sample=s,sourcePath=CONTROLS) for s in json.loads((ROOT/CONTROLS).read_text())]
 return [('baseline',controls),('candidate',historical+controls)]
if __name__=='__main__':executor.execute(HERE,pipeline,plans(),dict(pipeline='tone-chain-v1',sourceCases=425,plannedSourceAttempts=449,newAsrGenerations=0,newOcrGenerations=0,scope='Actual cleanup/conditional selection/latest formatting for candidate401 history+24 controls; baseline24 controls separately executed. Full typed context, no new audio/ASR/Vision/native insertion/concurrent/min16/production/fresh24 or4B/olderQwen evidence. No priming; startup separate, quality experiment not latency-SLO comparison.'))
