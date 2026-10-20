"""Test exact actual-OCR binding and complete review coverage."""
import json,pathlib,shutil,tempfile,unittest
import report
class EvidenceTests(unittest.TestCase):
    def test_complete(self):
        self.assertEqual(len(report.checked(report.RUN/'dev','dev')[1]),52)
        self.assertEqual(len(report.checked(report.RUN/'vision','ocr')[1]),48)
    def test_changed_ocr_context(self):
        with tempfile.TemporaryDirectory() as name:
            out=pathlib.Path(name)/'vision';shutil.copytree(report.RUN/'vision',out)
            rows=[json.loads(x) for x in (out/'results.jsonl').read_text().splitlines()];rows[0]['context']['Screen']='changed OCR'
            (out/'results.jsonl').write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in rows))
            manifest=json.loads((out/'manifest.json').read_text());manifest['resultsSha256']=report.engine.sha(out/'results.jsonl');(out/'manifest.json').write_text(json.dumps(manifest))
            review=json.loads((out/'review.json').read_text());review['resultsSha256']=manifest['resultsSha256']
            for item in review['reviews']:
                if (item['variant'],item['id'])==(rows[0]['variant'],rows[0]['id']):item['rowSha256']=report.rh(rows[0])
            (out/'review.json').write_text(json.dumps(review))
            with self.assertRaisesRegex(ValueError,'actual OCR context'):report.checked(out,'ocr')
    def test_missing_judgment(self):
        with tempfile.TemporaryDirectory() as name:
            out=pathlib.Path(name)/'dev';shutil.copytree(report.RUN/'dev',out);review=json.loads((out/'review.json').read_text());review['reviews'].pop();(out/'review.json').write_text(json.dumps(review))
            with self.assertRaisesRegex(ValueError,'Missing/duplicate'):report.checked(out,'dev')
    def test_portable_replay_uses_archived_source(self):
        frozen=report.check_frozen();mapping=report.engine.read(report.RUN/'path-mappings.json')
        self.assertTrue(all((report.ROOT/p).exists() for p in mapping['sourceSnapshots'].values()))
        self.assertIn('.workspace/rewrite-entity-evidence-v1/arion-ocr',mapping['externalArtifacts'])
        self.assertEqual(len(frozen['files']),len(set(frozen['files'])))
if __name__=='__main__':unittest.main()
