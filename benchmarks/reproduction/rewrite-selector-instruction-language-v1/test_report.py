"""Reject changed language treatments, model partitions and missing source outcomes."""
import copy
import json
import unittest
from unittest.mock import patch
import report


class LanguageEvidenceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.values = {}
        for key in report.MODEL_KEYS:
            rows, labels, targets = report.checked(key)
            directory = report.RUN/key
            cls.values[key] = [rows, report.read(directory/'manifest.json'), report.read(directory/'review.json'),
                               list(map(json.loads,(directory/'case-outcomes.jsonl').read_text().splitlines())),
                               report.read(directory/'selection.json')]

    def corrupt(self, operation):
        for key, original in self.values.items():
            values = copy.deepcopy(original)
            operation(*values)
            with self.assertRaises(ValueError):
                report.validate(*values, report.RUN/key, key)

    def test_complete_scope(self):
        self.assertEqual(sum(len(values[0]) for values in self.values.values()),404)
        self.assertEqual(sum(len(values[3]) for values in self.values.values()),1828)
        self.assertTrue(all(r['renderedBody'] is None for values in self.values.values() for r in values[0]))

    def test_missing_response_rejected(self):
        self.corrupt(lambda rows,*rest:rows.pop())

    def test_missing_skip_rejected(self):
        self.corrupt(lambda rows,manifest,review,outcomes,*rest:outcomes.pop())

    def test_false_model_partition_rejected(self):
        self.corrupt(lambda rows,manifest,*rest:manifest.update(modelKey='unknown'))

    def test_changed_user_data_rejected(self):
        self.corrupt(lambda rows,*rest:rows[0]['request']['messages'][-1].update(content='Changed source'))

    def test_changed_english_system_rejected(self):
        self.corrupt(lambda rows,*rest:rows[0]['request']['messages'][0].update(content='Unregistered instruction'))

    def test_false_exact_request_claim_rejected(self):
        self.corrupt(lambda rows,*rest:rows[0]['comparisonReference'].update(sameRequest=True))

    def test_false_body_credit_rejected(self):
        self.corrupt(lambda rows,*rest:rows[0].update(renderedBody='Invented body'))

    def test_false_adoption_rejected(self):
        self.corrupt(lambda rows,manifest,review,outcomes,decision:decision.update(qualified=True))

    def test_fresh_go_claim_rejected(self):
        self.corrupt(lambda rows,manifest,*rest:manifest.update(newGoPreparations=457))


if __name__=='__main__':
    unittest.main()
