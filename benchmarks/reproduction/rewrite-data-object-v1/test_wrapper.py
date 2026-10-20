"""Check complete context, field selection, identical controls and unfiltered output."""
import unittest,json
import wrapper
SAMPLES=json.loads((wrapper.HERE/'development.json').read_text())
class WrapperTests(unittest.TestCase):
 def test_lowercase_changes_only_current_reference_keys(self):
  for sample in SAMPLES:
   actual,encoding=wrapper.payload(sample,'qwen-lowercase');original,_=wrapper.prior.payload(sample,'qwen-private-spans')
   self.assertEqual(actual['messages'][:-1],original['messages'][:-1])
   if not sample['context']:self.assertEqual(actual,original)
   else:
    old=json.dumps(sample['context'],ensure_ascii=False,separators=(',',':'));new=json.dumps(wrapper.normalized(sample['context']),ensure_ascii=False,separators=(',',':'))
    self.assertEqual(actual['messages'][-1]['content'],original['messages'][-1]['content'].replace(old,new,1))
 def test_complete_object_source_and_context(self):
  for sample in SAMPLES:
   request,encoding=wrapper.payload(sample,'qwen-data-object');obj=json.loads(request['messages'][-1]['content'])
   self.assertEqual(obj['reference'],wrapper.normalized(sample['context']));self.assertEqual(obj['dictation'],encoding['encodedSource'])
   self.assertEqual(wrapper.decode(obj['dictation'],encoding),encoding['cleanedSource'])
 def test_all_example_user_turns_are_objects(self):
  for scene in ['work','email','prompt','chat']:
   sample=next(s for s in SAMPLES if s['scene']==scene);request,_=wrapper.payload(sample,'qwen-data-object')
   for message in request['messages']:
    if message['role']=='user':self.assertIn('dictation',json.loads(message['content']))
   self.assertIn('JSON 对象的 dictation 字段',request['messages'][0]['content'])
 def test_private_map_stays_private(self):
  sample=next(s for s in SAMPLES if s['id']=='lp-stress-multi');request,encoding=wrapper.payload(sample,'qwen-data-object');obj=json.loads(request['messages'][-1]['content'])
  self.assertNotIn('read cash',request['messages'][-1]['content']);self.assertEqual(len(obj['opaqueTokens']),2)
  self.assertIn('loadCache',request['messages'][-1]['content'])
 def test_no_output_filtering(self):
  request,encoding=wrapper.payload(SAMPLES[0],'qwen-data-object')
  for raw in ['', 'DONE', '<done>', 'unrelated facts']:self.assertEqual(wrapper.decode(raw,encoding),raw)
if __name__=='__main__':unittest.main()
