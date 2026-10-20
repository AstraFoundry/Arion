"""Prevent source labels and cross-language examples from overriding script evidence."""
import unittest,json
import pipeline
class RoutingTests(unittest.TestCase):
 def request(self,source,scene='chat'):
  return pipeline.format_payload(dict(scene=scene,language='wrong',context={'Screen':'must not enter'}),source,{'hypotheses':[]},{},[])[0]
 def test_latin_conversation_has_no_han(self):
  r=self.request('Please keep my words unchanged.')
  self.assertTrue(all(not pipeline.profile(m['content'])['hasHan'] for m in r['messages']))
 def test_han_demos_have_no_latin(self):
  r=self.request('请先核对，不要提交。')
  for m in r['messages'][1:-1]:
   text=json.loads(m['content'])['source'] if m['role']=='user' else m['content'];self.assertFalse(pipeline.profile(text)['hasLatin'])
 def test_long_code_does_not_force_english(self):self.assertEqual(pipeline.route('先查 veryLongFunctionName，不要提交。'),'mixed')
 def test_all_scenes_have_single_script_examples(self):
  for scene in ['work','email','prompt','chat']:
   self.request('请核对，不要提交。',scene);self.request('Please check; do not submit.',scene)
 def test_extended_han_and_input_identity(self):
  self.assertEqual(pipeline.route('\U00020000'),'han');r=self.request('English source.');self.assertEqual(json.loads(r['messages'][-1]['content'])['source'],'English source.')
if __name__=='__main__':unittest.main()
