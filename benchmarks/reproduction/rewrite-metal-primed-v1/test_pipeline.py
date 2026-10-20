"""Keep prefix priming general and separate from measured source material."""
import json,pathlib,unittest
import pipeline
class PrimingInputTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):cls.samples=json.loads((pipeline.HERE/'warmups.json').read_text());cls.inputs=json.loads((pipeline.HERE/'inputs.json').read_text())
 def test_three_routes_and_selector_each_scene(self):
  for scene in ['work','email','prompt','chat']:
   rows=[s for s in self.samples if s['workerScene']==scene];self.assertEqual(len(rows),4);self.assertEqual(sum(s['stage']=='selection' for s in rows),1);self.assertEqual({pipeline.prior.prior.formatter.route(s['source']) for s in rows if s['stage']=='format'},{'han','latin','mixed'})
 def test_no_marker_in_measured_material(self):
  text=json.dumps(self.inputs,ensure_ascii=False)
  for s in self.samples:
   if s['canary']:self.assertNotIn(s['canary'],text);self.assertIn(s['canary'],s['source'])
if __name__=='__main__':unittest.main()
