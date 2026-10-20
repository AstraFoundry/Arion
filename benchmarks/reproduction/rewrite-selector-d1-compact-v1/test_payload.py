"""Verify compact layouts keep every actual source and OCR field."""
import copy,json,unittest
import layout
class CompactTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):cls.records={k:json.loads((layout.HERE/k/'probe-inputs.json').read_text())['records'] for k in ['structured','sentence']}
 def test_both_layouts(self):
  for k,r in self.records.items():self.assertEqual(len(layout.validate(r,k)),6)
 def reject(self,k,change):
  r=copy.deepcopy(self.records[k]);change(r)
  with self.assertRaises(ValueError):layout.validate(r,k)
 def test_context_loss(self):self.reject('structured',lambda r:r[0]['request']['state'].update(context={}))
 def test_source_loss(self):self.reject('sentence',lambda r:r[0]['request']['state'].update(source=''))
 def test_rule_loss(self):self.reject('structured',lambda r:r[0]['request']['questions']['e0'].update(instructions='Keep'))
 def test_changed_candidate(self):self.reject('sentence',lambda r:r[0]['request']['questions']['e0']['criteria'].update(replace='Invented alternative'))
if __name__=='__main__':unittest.main()
