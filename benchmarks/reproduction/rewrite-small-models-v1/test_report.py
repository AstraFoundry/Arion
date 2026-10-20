"""Evidence checks must reject changed requests even with recomputed row hashes."""
import copy,json,pathlib,tempfile,unittest
import report
class EvidenceTests(unittest.TestCase):
 def test_complete_both_models(self):
  for model in report.MODELS:
   rows,judgments=report.checked(report.RUN/model);self.assertEqual(len(rows),24);self.assertEqual(len(judgments),24)
 def test_incomplete_review_rejected(self):
  self.mutate(lambda rows,review:review['reviews'].pop())
 def test_changed_request_rejected_after_rehash(self):
  def change(rows,review):rows[0]['request']['messages'][-1]['content']='Changed dictation'
  self.mutate(change,rehash=True)
 def test_wrong_model_rejected_after_rehash(self):
  def change(rows,review):rows[0]['modelSha256']='0'*64
  self.mutate(change,rehash=True)
 def test_duplicate_review_rejected(self):
  def change(rows,review):review['reviews'][1]=copy.deepcopy(review['reviews'][0])
  self.mutate(change)
 def mutate(self,fn,rehash=False):
  source=report.RUN/report.MODELS[0]
  with tempfile.TemporaryDirectory() as tmp:
   target=pathlib.Path(tmp)
   for name in ['manifest.json','review.json','results.jsonl']:(target/name).write_bytes((source/name).read_bytes())
   rows=[json.loads(x) for x in (target/'results.jsonl').read_text().splitlines()];review=report.read(target/'review.json');fn(rows,review)
   if rehash:
    review['reviews'][0]['rowSha256']=report.rowhash(rows[0]);(target/'results.jsonl').write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in rows));manifest=report.read(target/'manifest.json');manifest['resultsSha256']=report.sha(target/'results.jsonl');review['resultsSha256']=manifest['resultsSha256'];report.study.write(target/'manifest.json',manifest)
   report.study.write(target/'review.json',review)
   with self.assertRaises(ValueError):report.checked(target)
if __name__=='__main__':unittest.main()
