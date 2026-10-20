"""Check explicit user policy coverage and retained semantic failures."""
import unittest,review
class AlignmentTests(unittest.TestCase):
 def test_complete_source_bound_review(self):
  manifest,items,policy=review.checked();self.assertEqual(manifest['reviewedOutputs'],25);self.assertEqual(manifest['reclassifiedFaithful'],16);self.assertFalse(manifest['newInference']);self.assertEqual(policy['answer'],'允许规范成 readCash，但不能无依据改成 readCache')
 def test_target_replacement_still_fails(self):
  _,items,_=review.checked();self.assertTrue(all(not j['alignedFaithful'] for j in items['reviews'] if j['id']=='lp-stress-condition'))
 def test_good_normalization_retains_original_failure(self):
  _,items,_=review.checked();j=next(j for j in items['reviews'] if j['run']=='rewrite-small-models-v1/qwen3.5-2b' and j['id']=='p4-work-ambiguous');self.assertFalse(j['originalFaithful']);self.assertTrue(j['alignedFaithful'])
if __name__=='__main__':unittest.main()
