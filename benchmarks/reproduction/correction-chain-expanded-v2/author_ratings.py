"""Bind explicit agent review decisions to delivered output bytes, never infer scores."""
import argparse
import hashlib
import json
import pathlib

ROOT = pathlib.Path(__file__).resolve().parents[3]
ARMS = ['asr','asr_correct','asr_llm','asr_correct_llm','offline','offline_llm']

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('run')
    parser.add_argument('decisions')
    parser.add_argument('output')
    args = parser.parse_args()
    run = pathlib.Path(args.run)
    acoustic = {r['caseId']:r for r in map(json.loads,(run/'acoustic.jsonl').read_text().splitlines())}
    corrected = {r['caseId']:r for r in map(json.loads,(run/'correction.jsonl').read_text().splitlines())}
    rewrites = {r['caseId']:r for r in map(json.loads,(run/'rewrite.jsonl').read_text().splitlines())}
    decisions = json.loads(pathlib.Path(args.decisions).read_text())
    labels = json.loads((ROOT/'benchmarks/corpora/correction-chain-expanded-v2/labels.json').read_text())
    ratings = []
    for case, decision in decisions.items():
        assert len(decision['codes']) == 6
        for arm, code in zip(ARMS,decision['codes']):
            if arm.endswith('_llm'):
                record = rewrites.get(case+'--'+arm)
                text = record['text'] if record else ''
            else:
                text = corrected[case]['text'] if arm=='asr_correct' else acoustic[case]['offline' if arm=='offline' else 'streaming']['text']
            fidelity = {'P':'preserved','D':'degraded','U':'uncertain'}[code[0]]
            organization = int(code[1])
            reason = decision.get('reasons',{}).get(arm,decision.get('reasons',{}).get(code))
            if reason is None:
                assert code == 'P2', (case,arm,'Missing concrete rationale')
                reason = 'Clean wording preserves the final intent: '+labels[case]['references'][0]
            ratings.append(dict(caseId=case,arm=arm,textSha256=hashlib.sha256(text.encode()).hexdigest(),
                fidelity=fidelity,organization=organization,ready=fidelity=='preserved' and organization>=1,reason=reason))
    value = dict(kind='agent-semantic-review-expanded-v2',runSha256=sha(run/'run.json'),
        rubricSha256=sha(ROOT/'benchmarks/reproduction/correction-chain-semantic-v1/rubric.json'),
        decisionSha256=sha(pathlib.Path(args.decisions)),reviewer='Codex single-agent authored semantic review',
        independence='Non-blinded agent review after inference; no independent human adjudication or inter-rater reliability.',ratings=ratings)
    with pathlib.Path(args.output).open('x') as output:
        json.dump(value,output,ensure_ascii=False,indent=2)
        output.write('\n')

if __name__ == '__main__':
    main()
