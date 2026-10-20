"""Keep language controls source-derived and retain complete visible data."""
import unittest,json
import pipeline
class ProfileTests(unittest.TestCase):
 def test_english_only(self):self.assertEqual(pipeline.profile("You're welcome."),dict(hasHan=False,hasLatin=True))
 def test_chinese_with_long_code_is_mixed(self):self.assertEqual(pipeline.profile('先查 veryLongFunctionIdentifier，别提交。'),dict(hasHan=True,hasLatin=True))
 def test_extended_han(self):self.assertEqual(pipeline.profile('\U00020000'),dict(hasHan=True,hasLatin=False))
 def test_punctuation_does_not_invent_english(self):self.assertEqual(pipeline.profile('你好，四秒。'),dict(hasHan=True,hasLatin=False))
 def test_no_reference_or_language_label_used(self):
  s=dict(id='arbitrary',scene='chat',source='Only the source.',context={'Screen':'must not reach formatter'},language='zh');request,e=pipeline.format_payload(s,s['source'],{'hypotheses':[]},{},[]);last=json.loads(request['messages'][-1]['content']);self.assertEqual(last,dict(source=s['source'],sourceScripts=dict(hasHan=False,hasLatin=True)));self.assertNotIn('Screen',request['messages'][-1]['content'])
if __name__=='__main__':unittest.main()
