"""Reject changed ASR source or speaker provenance even when parent files are intact."""
import json,pathlib,shutil,tempfile,unittest
import verify
class SourceAuditTests(unittest.TestCase):
 def test_complete(self):self.assertEqual(verify.checked()['eligibleNaturalSources'],120)
 def alter(self,filename,field):
  with tempfile.TemporaryDirectory() as directory:
   target=pathlib.Path(directory)
   for p in verify.HERE.iterdir():
    if p.is_file():shutil.copyfile(p,target/p.name)
   rows=json.loads((target/filename).read_text());rows[0][field]='fabricated';(target/filename).write_text(json.dumps(rows,ensure_ascii=False))
   with self.assertRaises(ValueError):verify.checked(target)
 def test_source_changed(self):self.alter('inputs.json','source')
 def test_speaker_changed(self):self.alter('evaluator.json','sourceSpeakerId')
if __name__=='__main__':unittest.main()
