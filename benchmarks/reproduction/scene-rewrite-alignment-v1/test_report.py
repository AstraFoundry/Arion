"""Integrity tests use real archived rows, never fabricated inference results."""
import hashlib
import json
import pathlib
import shutil
import tempfile
import unittest

import report
import study

class EvidenceIntegrity(unittest.TestCase):
    def copy_run(self):
        tmp=tempfile.TemporaryDirectory();self.addCleanup(tmp.cleanup)
        path=pathlib.Path(tmp.name)/'dev'
        shutil.copytree(study.ROOT/'benchmarks/runs/scene-rewrite-alignment-v1/dev',path)
        return path
    def mutate_rows(self,path,transform):
        rows=[json.loads(l) for l in (path/'audited.jsonl').read_text().splitlines()]
        rows=transform(rows)
        (path/'audited.jsonl').write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in rows))
        m=study.read(path/'manifest.json');m['auditedSha256']=study.sha(path/'audited.jsonl');study.write(path/'manifest.json',m)
    def test_original_replays(self):
        _,samples,rows,_=report.load_run(self.copy_run(),'dev')
        self.assertEqual((len(samples),len(rows)),(24,48))
    def test_partial_run_is_not_reportable(self):
        path=self.copy_run();m=study.read(path/'manifest.json');m['status']='failed';study.write(path/'manifest.json',m)
        with self.assertRaisesRegex(ValueError,'Incomplete'):report.load_run(path,'dev')
    def test_missing_case_rejected_after_hash_recomputed(self):
        path=self.copy_run();self.mutate_rows(path,lambda rows:rows[:-1])
        with self.assertRaisesRegex(ValueError,'Incomplete'):report.load_run(path,'dev')
    def test_duplicate_case_rejected_after_hash_recomputed(self):
        path=self.copy_run();self.mutate_rows(path,lambda rows:rows[:-1]+[rows[0]])
        with self.assertRaisesRegex(ValueError,'duplicate'):report.load_run(path,'dev')
    def test_changed_candidate_invalidates_hash(self):
        path=self.copy_run()
        with (path/'audited.jsonl').open('a') as f:f.write('\n')
        with self.assertRaisesRegex(ValueError,'Changed audited'):report.load_run(path,'dev')
    def test_changed_verifier_decision_rejected(self):
        path=self.copy_run()
        def alter(rows):
            row=next(r for r in rows if r['envelope']=='accepted')
            row['delivery']['semantic']='invented-success'
            return rows
        self.mutate_rows(path,alter)
        with self.assertRaisesRegex(ValueError,'semantic delivery'):report.load_run(path,'dev')
    def test_changed_score_rejected(self):
        path=self.copy_run()
        def alter(rows):rows[0]['rawConstraintScore']['factsPassed']=not rows[0]['rawConstraintScore']['factsPassed'];return rows
        self.mutate_rows(path,alter)
        with self.assertRaisesRegex(ValueError,'constraint score'):report.load_run(path,'dev')

if __name__=='__main__':unittest.main()
