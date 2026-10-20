"""Exercise missing review, changed request and reviewed output guards."""
import copy
import json
import pathlib
import shutil
import tempfile
import unittest
import report
ROOT=report.study.ROOT
RUN=ROOT/'benchmarks/runs/rewrite-packaging-v1'
class EvidenceTests(unittest.TestCase):
    def test_complete(self):
        self.assertEqual(len(report.checked(RUN/'dev','dev')[1]),54)
        self.assertEqual(len(report.checked(RUN/'evaluation','evaluation','labeled')[1]),96)
    def corrupt_review(self,mutate):
        with tempfile.TemporaryDirectory() as name:
            target=pathlib.Path(name)/'dev';shutil.copytree(RUN/'dev',target)
            review=json.loads((target/'review.json').read_text());mutate(review)
            (target/'review.json').write_text(json.dumps(review))
            with self.assertRaises(ValueError):report.checked(target,'dev')
    def test_missing_review(self):self.corrupt_review(lambda r:r['reviews'].pop())
    def test_changed_output_binding(self):self.corrupt_review(lambda r:r['reviews'][0].update(rowSha256='changed'))
    def test_duplicate_review(self):self.corrupt_review(lambda r:r['reviews'].append(copy.deepcopy(r['reviews'][0])))
    def test_missing_repair_rating(self):
        def remove(r):del r['reviews'][0]['repairSucceeded']
        # Missing keys must fail explicitly rather than default to success.
        with self.assertRaises(KeyError):self.corrupt_review(remove)
if __name__=='__main__':unittest.main()
