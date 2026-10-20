"""Check complete review coverage and tamper detection without model inference."""
import copy
import hashlib
import json
import pathlib
import tempfile
import unittest
import report

ARCHIVE=report.ROOT/'benchmarks/runs/scene-rewrite-prompt-v2'

class ReviewIntegrity(unittest.TestCase):
    def setUp(self):
        self.directory=ARCHIVE/'evaluation'
        self.manifest,self.rows=report.checked_run(self.directory,'evaluation')
        self.review=report.engine.read(self.directory/'review.json')

    def check_review(self,value):
        with tempfile.TemporaryDirectory() as temporary:
            path=pathlib.Path(temporary)/'review.json'
            report.engine.write(path,value)
            return report.checked_review(path,self.rows,self.directory/'results.jsonl')

    def test_complete_archive(self):
        self.assertEqual(len(self.check_review(self.review)),112)
        _,development=report.checked_run(ARCHIVE/'dev','dev')
        self.assertEqual(len(development),32)

    def test_missing_review(self):
        value=copy.deepcopy(self.review);value['reviews'].pop()
        with self.assertRaisesRegex(ValueError,'Incomplete review'):self.check_review(value)

    def test_duplicate_review(self):
        value=copy.deepcopy(self.review);value['reviews'].append(value['reviews'][0])
        with self.assertRaisesRegex(ValueError,'duplicate review'):self.check_review(value)

    def test_changed_candidate(self):
        self.rows[0]['response']['text']='Changed output'
        with self.assertRaisesRegex(ValueError,'mismatch'):self.check_review(self.review)

    def test_missing_generation_with_updated_hash(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory=pathlib.Path(temporary)
            data='\n'.join(json.dumps(r) for r in self.rows[:-1])+'\n'
            (directory/'results.jsonl').write_text(data)
            manifest=copy.deepcopy(self.manifest)
            manifest['resultsSha256']=hashlib.sha256(data.encode()).hexdigest()
            report.engine.write(directory/'manifest.json',manifest)
            with self.assertRaisesRegex(ValueError,'Missing, duplicated'):report.checked_run(directory,'evaluation')

    def test_raw_hash_changed(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory=pathlib.Path(temporary)
            (directory/'results.jsonl').write_text('{}\n')
            report.engine.write(directory/'manifest.json',self.manifest)
            with self.assertRaisesRegex(ValueError,'Changed raw'):report.checked_run(directory,'evaluation')

if __name__=='__main__':unittest.main()
