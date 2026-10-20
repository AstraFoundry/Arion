"""Require complete source, OCR, demonstrations and reversible data decoding."""
import copy,json,unittest
import layout
class LayoutTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):cls.records={k:json.loads((layout.HERE/k/'probe-inputs.json').read_text())['records'] for k in ['structured','sentence']}
 def test_complete_layouts(self):
  for kind,records in self.records.items():self.assertEqual(len(layout.validate(records,kind)),6)
 def reject(self,kind,change):
  records=copy.deepcopy(self.records[kind]);change(records)
  with self.assertRaises(ValueError):layout.validate(records,kind)
 def test_ocr_loss(self):self.reject('structured',lambda r:r[0]['request']['state'].update(context={}))
 def test_missing_demonstration(self):self.reject('structured',lambda r:r[0]['request']['state']['demonstrations'].pop())
 def test_changed_demonstration_answer(self):self.reject('sentence',lambda r:r[0]['request']['state']['demonstrations'][0].update(choices={'e0':'keep'}))
 def test_changed_original_sentence(self):self.reject('sentence',lambda r:r[0]['request']['questions']['e0']['criteria'].update(keep='Invented sentence'))
 def test_missing_trusted_rule(self):self.reject('structured',lambda r:r[0]['request']['questions']['e0'].update(instructions='Choose replace'))
if __name__=='__main__':unittest.main()
