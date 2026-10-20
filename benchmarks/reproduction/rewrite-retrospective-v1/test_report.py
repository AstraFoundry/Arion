"""Check that derived historical views cannot change original inference."""
import json,pathlib,shutil,tempfile,unittest
from unittest import mock
import report
class HistoricalEvidenceTests(unittest.TestCase):
    def test_complete(self):
        self.assertEqual(len(report.checked('dev')[0]),48)
        self.assertEqual(len(report.checked('test')[0]),48)
    def test_rehashed_candidate_still_must_match_original(self):
        with tempfile.TemporaryDirectory() as name:
            out=pathlib.Path(name)/'review';shutil.copytree(report.OUT/'dev',out/'dev')
            rows=[json.loads(x) for x in (out/'dev/results.jsonl').read_text().splitlines()];rows[0]['response']['text']='changed historical output';(out/'dev/results.jsonl').write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in rows))
            manifest=json.loads((out/'dev/manifest.json').read_text());manifest['resultsSha256']=report.sha(out/'dev/results.jsonl');(out/'dev/manifest.json').write_text(json.dumps(manifest))
            review=json.loads((out/'dev/review.json').read_text());review['resultsSha256']=manifest['resultsSha256'];review['reviews'][0]['rowSha256']=report.rh(rows[0]);(out/'dev/review.json').write_text(json.dumps(review))
            with mock.patch.object(report,'OUT',out):
                with self.assertRaisesRegex(ValueError,'Historical inference changed'):report.checked('dev')
if __name__=='__main__':unittest.main()
