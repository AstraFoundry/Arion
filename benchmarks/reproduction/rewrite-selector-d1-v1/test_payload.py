"""Reject source loss before sending native typed decisions."""
import copy,json,unittest
from request_contract import HERE,validate
class PayloadTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):cls.records=json.loads((HERE/'probe-inputs.json').read_text())['records']
 def reject(self,operation):
  value=copy.deepcopy(self.records);operation(value)
  with self.assertRaises(ValueError):validate(value)
 def test_complete_original_messages(self):self.assertEqual(len(validate(self.records)),6)
 def test_missing_example_rejected(self):self.reject(lambda r:r[0]['request']['state']['originalMessages'].pop(2))
 def test_context_loss_rejected(self):self.reject(lambda r:r[0]['request']['state']['originalMessages'][-1].update(content='{}'))
 def test_missing_question_rejected(self):self.reject(lambda r:r[0]['request'].update(questions={}))
 def test_changed_choice_labels_rejected(self):self.reject(lambda r:r[0]['request']['questions']['e0'].update(criteria={'yes':'yes','no':'no'}))
 def test_missing_input_rejected(self):self.reject(lambda r:r.pop())
 def test_changed_source_identity_rejected(self):self.reject(lambda r:r[0]['sourceReference'].update(rowSha256='bad'))
if __name__=='__main__':unittest.main()
