"""Read complete source/context/output tuples in bounded ranges."""
from pathlib import Path
import json,sys
D=Path(__file__).resolve().parent;rows=[json.loads(l) for l in (D/'run/results.jsonl').read_text().splitlines()]
start,end=map(int,sys.argv[1:3])
for n,r in enumerate(rows[start:end],start):
 print(json.dumps(dict(index=n,arm=r['arm'],id=r['id'],scene=r['scene'],source=r['source'],context=r['context'],hypotheses=[dict(sourceSpan=h['sourceSpan'],candidateSpellings=h['candidateSpellings']) for h in r['prepared']['hypotheses']],raw=r['response']['choices'][0]['message']['content'],stop=r['response']['choices'][0]['finish_reason'],edited=r['editedSource'],error=r['renderError']),ensure_ascii=False))
print('READ RANGE',start,min(end,len(rows)),'of',len(rows))
