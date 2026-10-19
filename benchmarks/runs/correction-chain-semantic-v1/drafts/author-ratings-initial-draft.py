"""Explicit agent-authored judgments after reading all 156 delivered outputs.

These are semantic annotations, not predictions from reference matching. The
reviewer knew the model identities. Every rating is bound to the exact output.
"""
import hashlib
import json
import pathlib

ROOT = pathlib.Path(__file__).resolve().parents[3]
ARMS = ['asr', 'asr_correct', 'asr_llm', 'asr_correct_llm', 'offline', 'offline_llm']
# Per-case manual labels: P=preserved, D=degraded, U=uncertain; digit=organization.
JUDGMENTS = {
    'test-zh-00': ('P0 P0 P0 P0 P0 P0', 'The final nine lunches remains explicit, but every output retains the superseded twelve and spoken repair.'),
    'test-zh-01': ('D1 D1 D2 D1 P0 P2', 'The first four outputs say Wang rather than Chen; the offline source preserves the repair and its rewrite correctly selects Chen.'),
    'test-zh-02': ('P0 P0 D1 P1 P0 D1', 'The direct and offline rewrites reverse the final day to Thursday. The corrected rewrite repeats Friday but preserves its intended afternoon; repetition reduces polish, not fidelity.'),
    'test-zh-03': ('P0 P0 P0 P2 P0 P2', 'All retain the final 7500 amount. The corrected and offline rewrites resolve the obsolete 6000 and repair; the other outputs still need cleanup.'),
    'test-zh-04': ('P1 P2 P1 P2 P2 P2', 'The backup prohibition and temporary-file action remain recoverable. The homophonic zhi typo in two outputs is a surface defect; the corrected and offline variants express the restriction clearly.'),
    'test-zh-05': ('P2 P2 P2 P2 P2 P2', 'The permissions hypothesis and uncertainty are preserved; punctuation and spacing are immaterial.'),
    'test-zh-06': ('P0 P0 P0 P0 P0 P1', 'The plan and suggestion to discuss remain. The malformed leading an-na-ge phrase in the streaming variants needs cleanup; the offline rewrite has only a minor residual filler.'),
    'test-zh-07': ('P2 P2 P2 P2 P2 P2', 'The reimbursement materials and next-Monday deadline are unchanged.'),
    'test-zh-08': ('P0 P0 P2 P2 P0 P2', 'Teacher Zhao and the request to adjust defense order are preserved. Inline clean prose and email paragraphs are both acceptable; raw fillers reduce readiness.'),
    'test-zh-09': ('P1 P1 P2 P2 P1 P2', 'All three checks are present in the same order. Enumerated prose is adequate; numbered lines improve structure without being required.'),
    'test-zh-10': ('P2 P2 P2 P2 P2 P2', 'All delivered outputs preserve the dictated instruction as text. A rejected/truncated raw model proposal does not invalidate a faithful delivered source.'),
    'test-zh-11': ('P2 P2 P2 P2 P2 P2', 'Both the denial of publication consent and absence of a delivery commitment are preserved.'),
    'test-en-12': ('D0 D2 D1 D2 P0 P0', 'Streaming outputs lose the final eight: aid is not a recoverable quantity, and correction selects twelve. Offline outputs retain the explicit final eight but leave the spoken repair.'),
    'test-en-13': ('P0 D2 P0 D2 P0 P0', 'The correction stages discard Robert and choose the superseded Helen. Other outputs preserve the final recipient but retain the spoken repair.'),
    'test-en-14': ('P2 P2 P2 P2 P2 P2', 'The prohibition and backup-completion condition are preserved.'),
    'test-en-15': ('D2 D2 D2 D2 D2 D2', 'Every output changes cache to cash; the hypothesized cause changes materially despite preserving uncertainty.'),
    'test-en-16': ('P0 P0 P2 P0 P0 P2', 'The proposal to postpone the release remains. Raw fillers and the corrected huh introduction need cleanup; direct/offline rewrites are clean.'),
    'test-en-17': ('P0 P0 P2 P2 P0 P2', 'Recipient Nora, receipt of the draft and review tomorrow remain. Dropping optional thanks and changing capitalization are permitted; the corrected email is semantically acceptable.'),
    'test-en-18': ('P1 P1 P2 P1 P1 P2', 'Charger, notebook and passport are retained. Clear first/second/third prose is accepted even without numbered lines.'),
    'test-en-19': ('P2 P2 P2 P2 P2 P2', 'Deployment failure and the expired-certificate cause are preserved.'),
    'test-mixed-20': ('D0 D0 D1 D1 U0 U1', 'The streaming variants replace PostgreSQL with generic post. The offline poster S Q L spelling suggests the intended database but is insufficiently reliable to certify its identity.'),
    'test-mixed-21': ('P1 P2 P1 P2 P1 P1', 'A P I and Mac O S preserve the same letters and platform identities; spacing/casing does not change the Windows-failure/macOS-success contrast.'),
    'test-mixed-22': ('P0 P0 P1 P1 U0 U1', 'In review context, pool request is an unambiguous minor spelling artifact for pull request; lower presentation quality, not automatic semantic failure. Good request in offline outputs leaves the technical object ambiguous.'),
    'test-mixed-23': ('D2 D2 D2 D2 P2 P2', 'Doclear does not reliably identify Docker. Offline recognition and its rewrite preserve Docker, configuration retention and the port prohibition.'),
    'test-silence-0': ('P2 P2 P2 P2 P2 P2', 'All actual delivered outputs are empty; nonempty rejected model proposals are recorded separately.'),
    'test-silence-1': ('P2 P2 P2 P2 P2 P2', 'All actual delivered outputs are empty; nonempty rejected model proposals are recorded separately.'),
}


def digest(data):
    return hashlib.sha256(data).hexdigest()


def main():
    output = ROOT / 'benchmarks/results/correction-chain-semantic-v1/ratings.json'
    summary_path = ROOT / 'benchmarks/results/correction-chain-v1/summary.json'
    rubric_path = pathlib.Path(__file__).with_name('rubric.json')
    summary = json.loads(summary_path.read_text())
    rows = {(r['caseId'], r['arm']): r for r in summary['rows']}
    if len(rows) != 156 or set(case for case, _ in rows) != set(JUDGMENTS):
        raise ValueError('Unexpected review coverage')
    ratings = []
    for case, (judgments, reason) in JUDGMENTS.items():
        values = judgments.split()
        if len(values) != len(ARMS):
            raise ValueError('Every arm must have an explicit manual rating')
        for arm, judgment in zip(ARMS, values):
            row = rows[(case, arm)]
            fidelity = {'P': 'preserved', 'D': 'degraded', 'U': 'uncertain'}[judgment[0]]
            organization = int(judgment[1])
            ratings.append(dict(caseId=case, arm=arm, textSha256=digest(row['text'].encode()),
                fidelity=fidelity, organization=organization, ready=fidelity=='preserved' and organization>=1,
                reason=reason))
    value = dict(version=1, kind='agent-semantic-review-v1', reviewer='Codex agent; aware of original model identities and previous scores',
        independence='single non-blinded agent review; not independent human ratings',
        summarySha256=digest(summary_path.read_bytes()), rubricSha256=digest(rubric_path.read_bytes()),
        annotationSourceSha256=digest(pathlib.Path(__file__).read_bytes()), ratings=ratings)
    with output.open('x') as target:
        json.dump(value, target, ensure_ascii=False, indent=2)
        target.write('\n')


if __name__ == '__main__':
    main()
