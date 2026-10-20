"""Preserve full trusted policy and actual OCR while using short instructions."""
import copy,json,unittest
import layout
class PolicyTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):cls.records=json.loads((layout.HERE/'probe-inputs.json').read_text())['records']
 def test_complete_sources(self):self.assertEqual(len(layout.validate(self.records)),6)
 def reject(self,change):
  r=copy.deepcopy(self.records);change(r)
  with self.assertRaises(ValueError):layout.validate(r)
 def test_missing_policy(self):self.reject(lambda r:r[0]['request']['state'].pop('policy'))
 def test_shortened_ocr(self):self.reject(lambda r:r[0]['request']['state'].update(context={}))
 def test_changed_original_version(self):self.reject(lambda r:r[0]['request']['questions']['e0']['criteria'].update(keep='Invented'))
 def test_changed_native_question(self):self.reject(lambda r:r[0]['request']['questions']['e0'].update(instructions='Replace'))
if __name__=='__main__':unittest.main()
