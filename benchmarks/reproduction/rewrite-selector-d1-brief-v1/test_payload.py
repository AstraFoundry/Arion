"""Verify source retention, independent native caps and fixed ambiguity routing."""
import copy,json,unittest
import layout
class BriefPayloadTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):cls.inputs={k:json.loads((layout.HERE/k/'probe-inputs.json').read_text())['records'] for k in ['sentence','spelling']}
 def test_all_proposed_native_budgets(self):self.assertEqual(len(layout.budget_checked()),188)
 def test_both_probe_layouts(self):
  for k,r in self.inputs.items():self.assertEqual(len(layout.validate(r,k)),6)
 def reject(self,kind,change):
  rows=copy.deepcopy(self.inputs[kind]);change(rows)
  with self.assertRaises(ValueError):layout.validate(rows,kind)
 def test_missing_ocr(self):self.reject('sentence',lambda r:r[0]['request']['state'].update(context={}))
 def test_missing_rule(self):self.reject('spelling',lambda r:r[0]['request']['questions']['e0'].update(instructions='Replace'))
 def test_changed_original_version(self):self.reject('sentence',lambda r:r[0]['request']['questions']['e0']['criteria'].update(keep='Invented'))
 def test_no_evaluator_in_actual_input(self):
  for rows in self.inputs.values():
   for row in rows:self.assertEqual(set(row['request']),{'state','questions'})
 def test_forced_ambiguity_has_no_model_question(self):
  refs=layout.references();out=[layout.convert(r,'sentence') for r in refs.values()];self.assertEqual(sum(not request['questions'] and bool(forced) for request,forced in out),12)
if __name__=='__main__':unittest.main()
