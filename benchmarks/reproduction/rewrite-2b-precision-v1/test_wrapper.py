"""Confirm all36 Q8 requests/encodings match the pinned Q4 baseline exactly."""
import unittest,json,pathlib,hashlib
import wrapper
ROOT=wrapper.HERE.parents[2]
class WrapperTests(unittest.TestCase):
 def test_all36_requests_and_encodings_equal(self):
  baseline={r['id']:r for r in map(json.loads,(ROOT/'benchmarks/runs/rewrite-private-spans-v1/qwen-private-spans/results.jsonl').read_text().splitlines())}
  samples=json.loads((wrapper.HERE/'development.json').read_text());self.assertEqual(len(samples),36)
  for sample in samples:
   request,encoding=wrapper.payload(sample,'qwen-q8-private');self.assertEqual(request,baseline[sample['id']]['request']);self.assertEqual(encoding,baseline[sample['id']]['encoding'])
if __name__=='__main__':unittest.main()
