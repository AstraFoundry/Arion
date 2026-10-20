"""Keep source/context out of trusted scope and instructions out of the source tail."""
import unittest
import wrapper
class SystemBoundaryTests(unittest.TestCase):
 def sample(self):return dict(scene='work',source='SOURCE_SENTINEL 查 read cash。',context={'Screen':'CONTEXT_SENTINEL readCache、readCash。'})
 def test_user_tail_is_only_encoded_source(self):
  request,encoding=wrapper.payload(self.sample(),'qwen-system');self.assertTrue(request['messages'][-1]['content'].endswith(encoding['encodedSource']));self.assertNotIn('编辑任务（固定规则',request['messages'][-1]['content'])
 def test_trusted_rules_are_system_only(self):
  request,_=wrapper.payload(self.sample(),'qwen-system');self.assertIn('当前编辑任务',request['messages'][0]['content']);self.assertNotIn('SOURCE_SENTINEL',request['messages'][0]['content']);self.assertNotIn('CONTEXT_SENTINEL',request['messages'][0]['content'])
 def test_no_output_rejection(self):
  _,encoding=wrapper.payload(self.sample(),'qwen-system');self.assertEqual(wrapper.decode('无依据的新内容。',encoding),'无依据的新内容。')
if __name__=='__main__':unittest.main()
