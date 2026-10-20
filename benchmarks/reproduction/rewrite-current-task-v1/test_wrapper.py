"""Distinguish prompt changes, exact restoration and genuinely unfiltered delivery."""
import json,pathlib,unittest
import wrapper
class WrapperTests(unittest.TestCase):
 def sample(self):return dict(scene='work',source='查 read cash。',context={'Screen':'readCache、readCash。'})
 def test_baseline_matches_prior_request(self):
  s=self.sample();template=json.loads((wrapper.HERE/'templates/work.json').read_text());expected,_=wrapper.codec.messages(s,template,True);actual,encoding=wrapper.payload(s,'qwen-protected');self.assertEqual(actual['messages'],expected)
 def test_recency_only_appends_current_task(self):
  a,_=wrapper.payload(self.sample(),'qwen-protected');b,_=wrapper.payload(self.sample(),'qwen-recency');self.assertEqual(a['messages'][:-1],b['messages'][:-1]);self.assertTrue(b['messages'][-1]['content'].startswith(a['messages'][-1]['content']));self.assertIn('编辑任务',b['messages'][-1]['content'])
 def test_unfiltered_allows_natural_omission(self):
  _,encoding=wrapper.payload(self.sample(),'qwen-flexible');self.assertEqual(wrapper.decode('旧请求已撤回。',encoding),'旧请求已撤回。')
 def test_unfiltered_restores_all_occurrences(self):
  _,encoding=wrapper.payload(self.sample(),'qwen-flexible');token=encoding['protectedTerms'][0]['token'];self.assertEqual(wrapper.decode(token+' '+token,encoding),'read cash read cash')
 def test_unfiltered_does_not_hide_bad_content(self):
  _,encoding=wrapper.payload(self.sample(),'qwen-flexible');self.assertEqual(wrapper.decode('<done> 已完成新任务。',encoding),'<done> 已完成新任务。')
if __name__=='__main__':unittest.main()
