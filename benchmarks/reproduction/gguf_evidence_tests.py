"""Shared corruption tests for complete GGUF follow-up evidence."""
import json,pathlib,tempfile,unittest
import gguf_evidence as evidence
def suite(report,directory,count):
 class EvidenceTests(unittest.TestCase):
  def test_complete_review(self):
   rows,reviews=report.checked(directory);self.assertEqual(len(rows),count);self.assertEqual(len(reviews),count)
  def test_missing_review_rejected(self):self.mutate(lambda rows,review,manifest:review['reviews'].pop())
  def test_changed_request_rejected_after_rehash(self):
   def change(rows,review,manifest):rows[0]['request']['messages'][-1]['content']='Unexpected replacement'
   self.mutate(change,True)
  def test_wrong_model_command_rejected(self):
   def change(rows,review,manifest):manifest['workers'][0]['command'][2]='/tmp/unpinned.gguf'
   self.mutate(change)
  def test_wrong_scene_rejected_after_rehash(self):
   def change(rows,review,manifest):rows[0]['effectiveModelScene']='search'
   self.mutate(change,True)
  def mutate(self,change,rehash=False):
   with tempfile.TemporaryDirectory() as tmp:
    target=pathlib.Path(tmp)/'run';target.mkdir()
    for name in ['results.jsonl','review.json','manifest.json']:(target/name).write_bytes((directory/name).read_bytes())
    # Supplemental treatment evidence belongs to the run's parent.
    env=directory.parent/'run-environment.json'
    if env.exists():(target.parent/env.name).write_bytes(env.read_bytes())
    rows=[json.loads(l) for l in (target/'results.jsonl').read_text().splitlines()];review=evidence.read(target/'review.json');manifest=evidence.read(target/'manifest.json');change(rows,review,manifest)
    if rehash:
     review['reviews'][0]['rowSha256']=evidence.rowhash(rows[0]);(target/'results.jsonl').write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in rows));manifest['resultsSha256']=evidence.sha(target/'results.jsonl');review['resultsSha256']=manifest['resultsSha256']
    evidence.engine.write(target/'review.json',review);evidence.engine.write(target/'manifest.json',manifest)
    with self.assertRaises(ValueError):report.checked(target)
 return unittest.defaultTestLoader.loadTestsFromTestCase(EvidenceTests)
