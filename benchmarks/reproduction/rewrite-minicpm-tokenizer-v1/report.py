"""Replay exact official/native input token equality without inference or tokenizer code."""
import pathlib,json,hashlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
HERE=pathlib.Path(__file__).resolve().parent
RUN=ROOT/'benchmarks/runs/rewrite-minicpm-tokenizer-v1'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rowhash(x):return hashlib.sha256(json.dumps(x,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def checked():
    plan,status=read(RUN/'plan.json'),read(RUN/'status.json');raw=RUN/'native/tokenization.jsonl';rows=list(map(json.loads,raw.read_text().splitlines()))
    refs_path=ROOT/'benchmarks/runs/rewrite-selector-minicpm-v1/comparison-run/results.jsonl';refs=list(map(json.loads,refs_path.read_text().splitlines()))
    if status['status']!='complete' or status['requests']!=202 or status['identical']!=202 or status['modelGenerations']!=0 or len(rows)!=202 or status['rawSha256']!=sha(raw) or status['planSha256']!=sha(RUN/'plan.json') or plan['sourceResultsSha256']!=sha(refs_path) or plan['tokenizerSha256']!=sha(HERE/'tokenizer.json') or plan['tokenizersVersion']!='0.22.1':raise ValueError('Changed completed tokenizer evidence')
    if read(HERE/'receipt.json')['sha256']!=sha(HERE/'tokenizer.json'):raise ValueError('Changed pinned official tokenizer')
    for row,ref in zip(rows,refs):
        if (row['arm'],row['id'])!=(ref['arm'],ref['id']) or row['sourceRowSha256']!=rowhash(ref) or row['applyTemplateRequest']!=dict(messages=ref['request']['messages'],add_generation_prompt=True):raise ValueError('Changed actual complete request source')
        if row['tokenizeRequest']!=dict(content=row['applyTemplateResponse']['prompt'],add_special=False,parse_special=True,with_pieces=False) or row['tokenizeResponse']['tokens']!=row['upstreamTokenIds'] or row['identical'] is not True:raise ValueError('Changed exact native/official token IDs')
        if row['actualInferencePromptTokens']!=ref['response']['usage']['prompt_tokens']:raise ValueError('Changed inference reference')
    return dict(status='complete',completeRequests=202,exactTokenIdParity=202,modelGenerations=0,tokenizerVersion=plan['tokenizersVersion'],sourceModel='MiniCPM5-1B',scope='Actual native rendered prompts only; does not establish reference logits, converted weight numerical equivalence, semantic quality, independent release or actual16GB acceptance.')
if __name__=='__main__':print(json.dumps(checked(),indent=2))
