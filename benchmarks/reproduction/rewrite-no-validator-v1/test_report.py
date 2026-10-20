"""Check validator-free delivery semantics and review integrity without inference."""
import copy
import json
import pathlib
import shutil
import tempfile
import unittest
import report
from policies import delivery

class DeliveryAblation(unittest.TestCase):
    def row(self,strict='accepted',envelope='accepted',**response):
        data=dict(text='proposal',cleanup=dict(text='cleanup'));data.update(response)
        return dict(strict=strict,envelope=envelope,response=data)

    def test_content_rejection_is_removed(self):
        row=self.row(strict='unsupported_content')
        self.assertEqual(delivery(row,'strict')['text'],'cleanup')
        self.assertEqual(delivery(row,'envelope')['text'],'proposal')
        self.assertEqual(delivery(row,'none')['text'],'proposal')

    def test_none_delivers_protocol_and_truncation_unchanged(self):
        for verdict in ['markup','truncated']:
            row=self.row(strict=verdict,envelope=verdict,text='<dictation>partial',truncated=verdict=='truncated')
            self.assertEqual(delivery(row,'none')['text'],'<dictation>partial')
            self.assertEqual(delivery(row,'envelope')['origin'],'cleanup')

    def test_none_really_has_no_empty_text_gate(self):
        result=delivery(self.row(strict='empty',envelope='empty',text=''),'none')
        self.assertTrue(result['accepted']);self.assertEqual(result['text'],'')

    def test_deadline_is_not_overridden_by_none(self):
        row=self.row(applicationTimeout=True,cancelled=True,text='late partial')
        self.assertEqual(delivery(row,'none')['origin'],'cleanup')
        self.assertEqual(delivery(row,'none')['outcome'],'timeout')

    def test_cancelled_work_cannot_publish(self):
        row=self.row(cancelled=True)
        self.assertIsNone(delivery(row,'none')['text'])
        self.assertEqual(delivery(row,'none')['origin'],'suppressed')

class ReviewIntegrity(unittest.TestCase):
    def setUp(self):
        self.temporary=tempfile.TemporaryDirectory();self.addCleanup(self.temporary.cleanup)
        self.base=pathlib.Path(self.temporary.name)
        source=report.study.ROOT/'benchmarks/runs/rewrite-no-validator-v1'
        for name in ['dev','evaluation']:shutil.copytree(source/name,self.base/name)

    def test_every_exported_delivery_matches_its_raw_policy(self):
        archive=report.study.ROOT/'benchmarks/runs/rewrite-no-validator-v1'
        indexed={}
        for phase,directory in [('development','dev'),('evaluation','evaluation')]:
            for line in (archive/directory/'results.jsonl').read_text().splitlines():
                row=json.loads(line);indexed[(phase,row['variant'],row['id'])]=row
        records=[json.loads(line) for line in (archive/'report/deliveries.jsonl').read_text().splitlines()]
        keys=set()
        for record in records:
            key=(record['phase'],record['variant'],record['id'],record['policy'])
            self.assertNotIn(key,keys);keys.add(key)
            expected=delivery(indexed[key[:3]],record['policy'])
            for field,value in expected.items():self.assertEqual(record[field],value)
        self.assertEqual(len(keys),384)

    def test_complete(self):
        _,rows,reviews=report.checked(self.base/'evaluation','evaluation','anchored')
        self.assertEqual(len(rows),96);self.assertEqual(len(reviews),96)

    def mutate_review(self,change):
        path=self.base/'evaluation/review.json';data=json.loads(path.read_text());change(data);path.write_text(json.dumps(data))

    def test_missing_cleanup_review(self):
        self.mutate_review(lambda data:data['reviews'][0].update(cleanupFaithful=None))
        with self.assertRaisesRegex(ValueError,'Missing judgment'):report.checked(self.base/'evaluation','evaluation','anchored')

    def test_missing_review(self):
        self.mutate_review(lambda data:data['reviews'].pop())
        with self.assertRaisesRegex(ValueError,'Incomplete review'):report.checked(self.base/'evaluation','evaluation','anchored')

    def test_changed_reviewed_output(self):
        self.mutate_review(lambda data:data['reviews'][0].update(rowSha256='0'*64))
        with self.assertRaisesRegex(ValueError,'Changed reviewed'):report.checked(self.base/'evaluation','evaluation','anchored')

if __name__=='__main__':unittest.main()
