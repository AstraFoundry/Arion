"""A failed startup cannot be misreported as completed inference."""
import unittest
import report
class FailureEvidenceTests(unittest.TestCase):
 def test_explicit_zero_generation_failure(self):
  result=report.checked();self.assertEqual(result['status'],'failed-before-generation');self.assertEqual(result['generations'],0);self.assertFalse(result['productionActivated'])
if __name__=='__main__':unittest.main()
