"""Verify exact previews from real archived spans without any model inference."""
import copy,importlib.util,json,pathlib,unittest
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2]
s=importlib.util.spec_from_file_location('tested_preview_pipeline',HERE/'pipeline.py');pipeline=importlib.util.module_from_spec(s);s.loader.exec_module(pipeline)
class PreviewIntegrityTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):cls.rows=list(map(json.loads,(ROOT/'benchmarks/runs/rewrite-selector-semantic-v2/run/results.jsonl').read_text().splitlines()))
 def test_exact_english_source_letters(self):
  r=next(r for r in self.rows if r['id']=='tone-control-18' and r['stage']=='selection');data=pipeline.preview.augment(pipeline.original_data,r['prepared']);self.assertEqual(data['hypotheses'][0]['candidateRewrites'],[dict(spelling='readCash',sourceAfterReplacement='先查 readCash 的记录，别改配置。')])
 def test_wrong_domain_not_silently_removed(self):
  r=next(r for r in self.rows if r['id']=='role-deployment-3' and r['stage']=='selection');data=pipeline.preview.augment(pipeline.original_data,r['prepared']);self.assertEqual(data['hypotheses'][0]['candidateRewrites'][0]['sourceAfterReplacement'],'服务器部署的培植清单先核对，别执行部署。');self.assertEqual(data['context'],r['context'])
 def test_corrupted_span_rejected(self):
  p=copy.deepcopy(next(r['prepared'] for r in self.rows if r['id']=='tone-control-18' and r['stage']=='selection'));p['hypotheses'][0]['startUtf8']+=1
  with self.assertRaises(ValueError):pipeline.preview.augment(pipeline.original_data,p)
 def test_prepared_records_not_mutated(self):
  p=next(r['prepared'] for r in self.rows if r['id']=='history-0265' and r['stage']=='selection');before=copy.deepcopy(p);data=pipeline.preview.augment(pipeline.original_data,p);self.assertEqual(p,before);self.assertEqual(len(data['hypotheses']),2)
if __name__=='__main__':unittest.main()
