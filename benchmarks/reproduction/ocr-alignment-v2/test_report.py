"""Verify intended-message review and exact inherited text-policy delivery semantics."""
import json
import pathlib
import shutil
import tempfile
import unittest
import report

class ReviewIntegrity(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory();self.addCleanup(self.temp.cleanup)
        self.directory=pathlib.Path(self.temp.name)/'evaluation'
        shutil.copytree(report.study.ROOT/'benchmarks/runs/ocr-alignment-v2/evaluation',self.directory)

    def change_review(self,change):
        path=self.directory/'review.json';data=json.loads(path.read_text());change(data);path.write_text(json.dumps(data))

    def test_complete(self):
        _,rows,reviews=report.checked(self.directory,'evaluation','aligned')
        self.assertEqual(len(rows),96);self.assertEqual(len(reviews),96)

    def test_missing_review(self):
        self.change_review(lambda data:data['reviews'].pop())
        with self.assertRaisesRegex(ValueError,'Incomplete review'):report.checked(self.directory,'evaluation','aligned')

    def test_duplicate_review(self):
        self.change_review(lambda data:data['reviews'].append(data['reviews'][0]))
        with self.assertRaisesRegex(ValueError,'Duplicate'):report.checked(self.directory,'evaluation','aligned')

    def test_missing_repair_judgment(self):
        self.change_review(lambda data:data['reviews'][0].update(repairSucceeded=None))
        with self.assertRaisesRegex(ValueError,'Missing review judgment'):report.checked(self.directory,'evaluation','aligned')

    def test_changed_reviewed_output(self):
        self.change_review(lambda data:data['reviews'][0].update(rowSha256='0'*64))
        with self.assertRaisesRegex(ValueError,'Changed reviewed output'):report.checked(self.directory,'evaluation','aligned')

    def test_changed_raw(self):
        (self.directory/'results.jsonl').write_text('{}\n')
        with self.assertRaisesRegex(ValueError,'Changed experiment or observations'):report.checked(self.directory,'evaluation','aligned')

    def test_all_policy_exports_reproduce_exact_text(self):
        base=report.study.ROOT/'benchmarks/runs/ocr-alignment-v2';raw={}
        for phase,name in [('development','dev'),('evaluation','evaluation')]:
            for line in (base/name/'results.jsonl').read_text().splitlines():
                row=json.loads(line);raw[(phase,row['variant'],row['id'])]=row
        keys=set()
        for line in (base/'report/deliveries.jsonl').read_text().splitlines():
            item=json.loads(line);key=(item['phase'],item['variant'],item['id'],item['policy'])
            self.assertNotIn(key,keys);keys.add(key)
            expected=report.delivery(raw[key[:3]],item['policy'])
            for field,value in expected.items():self.assertEqual(item[field],value)
        self.assertEqual(len(keys),384)

    def test_complete_cold_forward_reverse_parity(self):
        directory=report.study.ROOT/'benchmarks/runs/ocr-alignment-v2/parity'
        _,rows,reviews,mismatches=report.checked_parity(directory)
        self.assertEqual(len(rows),12);self.assertEqual(len(reviews),12)
        self.assertEqual(mismatches,[])

    def test_missing_effective_scene_does_not_claim_automatic_detection(self):
        with self.assertRaisesRegex(ValueError,'Missing effective model scene'):
            report.scene_metadata(dict(scene='work',response=dict(template='test')))

if __name__=='__main__':unittest.main()
