"""Report every current response and explicitly reused paired baseline observation."""
import pathlib,importlib.util,sys,argparse,json
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-selector-semantic-v2/run'
def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
before=list(sys.path);sys.path.insert(0,str(HERE.parent))
try:evidence=module('semantic_v2_evidence',HERE.parent/'gguf_chain_evidence_v3.py')
finally:sys.path[:]=before
pipeline=module('semantic_v2_report_pipeline',HERE/'pipeline.py')
read=evidence.read;sha=evidence.sha;rowhash=evidence.rowhash
SOURCES=['benchmarks/corpora/rewrite-history-v1/inputs.json','benchmarks/corpora/rewrite-tone-controls-v1/inputs.json','benchmarks/corpora/rewrite-semantic-roles-v1/inputs.json']
def plans():return [('candidate',[dict(sample=s,sourcePath=p) for p in SOURCES for s in read(ROOT/p)])]
def validate(rows,m,j,o,d,directory=RUN):
 if d['qualified'] or d['selected'] is not None:raise ValueError('Rejected policy cannot be adopted')
 return evidence.validate(ROOT,HERE,pipeline,plans(),rows,m,j,o,d,directory)
def checked(directory=RUN):
 rows,labels=evidence.checked(ROOT,HERE,pipeline,plans(),directory)
 d=read(directory/'selection.json')
 if d['qualified'] or d['selected'] is not None:raise ValueError('Rejected policy cannot be adopted')
 comparison=read(directory/'reference-comparison.json')['records'];refs={};current={(r['stage'],r['id']):r for r in rows}
 for x in comparison:
  path=x['referencePath']
  if path not in refs:refs[path]={(r['arm'],r['stage'],r['id']):r for r in map(json.loads,(ROOT/path).read_text().splitlines())}
  r=current[(x['stage'],x['id'])];b=refs[path][(x['referenceArm'],x['stage'],x['id'])]
  if x['rowSha256']!=rowhash(r) or x['referenceRowSha256']!=rowhash(b) or x['sameBody']!=(r['response']['choices'][0]['message']['content']==b['response']['choices'][0]['message']['content']) or x['sameRequest']!=(r['request']==b['request']) or x['samePrepared']!=(r['prepared']==b['prepared']):raise ValueError('Changed paired baseline evidence')
 if len(comparison)!=2*len(rows) or len({(x['comparison'],x['stage'],x['id']) for x in comparison})!=2*len(rows):raise ValueError('Incomplete paired comparison')
 return rows,labels

def collect():
 rows,labels=checked();d=read(RUN/'selection.json')
 return dict(newGenerations=len(rows),sourceAttempts=457,groups=evidence.summarize(rows,labels),selection=d,limits=read(HERE/'protocol.json')['limits']),rows,labels

def main():
 p=argparse.ArgumentParser();p.add_argument('--out',type=pathlib.Path,required=True);a=p.parse_args();summary,rows,labels=collect();a.out.mkdir(parents=True,exist_ok=False);(a.out/'summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n');lines=['# Concise entity-positive semantic selector review','','Every actual current response personally reviewed. Reused baseline responses are compared by exact raw identity, not counted again as new inference. Both positive repairs and harmful replacements determine adoption; protocol validity alone is insufficient. All actual scenario assignments and complete requests retained.','']
 for r in rows:lines += [f"## {r['arm']} / {r['id']} / {r['scene']} / {r['stage']}",'','```json',json.dumps({k:r[k] for k in ['source','context','prepared','editedSource','edits','templateVersion','templatePath','templateSha256','actualMessagesSha256','request']},ensure_ascii=False,indent=2),'```','','```text',r['response']['choices'][0]['message']['content'],'```','',labels[(r['variant'],r['id'])]['note'],'']
 (a.out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(summary,ensure_ascii=False))
if __name__=='__main__':main()
