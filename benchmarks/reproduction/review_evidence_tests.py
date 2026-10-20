"""Shared adversarial integrity tests for complete reviewed experiment archives."""
import json,pathlib,shutil,tempfile,unittest
import review_evidence as evidence

def suite(module,directory,expected_count):
    class EvidenceTests(unittest.TestCase):
        def test_complete(self):self.assertEqual(len(module.checked(directory)[0]),expected_count)
        def copy(self):
            temporary=tempfile.TemporaryDirectory();self.addCleanup(temporary.cleanup)
            target=pathlib.Path(temporary.name)/'run';shutil.copytree(directory,target);return target
        def test_missing_review(self):
            target=self.copy();review=evidence.read(target/'review.json');review['reviews'].pop();(target/'review.json').write_text(json.dumps(review))
            with self.assertRaisesRegex(ValueError,'Incomplete/duplicate'):module.checked(target)
        def test_rehashed_model_identity(self):
            target=self.copy();rows=[json.loads(x) for x in (target/'results.jsonl').read_text().splitlines()];rows[0]['modelFiles']={'wrong-model':'wrong-hash'}
            self.rebind(target,rows)
            with self.assertRaisesRegex(ValueError,'actual model identity'):module.checked(target)
        def test_rehashed_template_identity(self):
            target=self.copy();rows=[json.loads(x) for x in (target/'results.jsonl').read_text().splitlines()];rows[0]['response']['template']='wrong-template'
            self.rebind(target,rows)
            with self.assertRaisesRegex(ValueError,'scene/template'):module.checked(target)
        def rebind(self,target,rows):
            (target/'results.jsonl').write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in rows));digest=evidence.sha(target/'results.jsonl');manifest=evidence.read(target/'manifest.json');manifest['resultsSha256']=digest;(target/'manifest.json').write_text(json.dumps(manifest))
            review=evidence.read(target/'review.json');review['resultsSha256']=digest
            for item in review['reviews']:
                if (item['variant'],item['id'])==(rows[0]['variant'],rows[0]['id']):item['rowSha256']=evidence.rowhash(rows[0])
            (target/'review.json').write_text(json.dumps(review))
    return unittest.defaultTestLoader.loadTestsFromTestCase(EvidenceTests)
