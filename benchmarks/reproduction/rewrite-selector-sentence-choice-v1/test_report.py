"""Reject incomplete choices, false normalization and counterbalance claims."""
import copy
import json
import unittest
from unittest.mock import patch
import report


class SentenceChoiceEvidenceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.rows, cls.labels = report.checked()
        cls.manifest, cls.review, cls.decision = [report.read(report.RUN/name) for name in ['manifest.json', 'review.json', 'selection.json']]
        cls.outcomes = list(map(json.loads, (report.RUN/'case-outcomes.jsonl').read_text().splitlines()))
        cls.preps = list(map(json.loads, (report.RUN/'preparations.jsonl').read_text().splitlines()))

    def corrupt(self, operation):
        values = copy.deepcopy([self.rows, self.manifest, self.review, self.outcomes, self.preps, self.decision])
        operation(*values)
        with self.assertRaises(ValueError):
            report.validate(*values, report.RUN)

    def corrupt_sidecar(self, filename, operation):
        original = report.read
        sidecar = original(report.RUN/filename)
        operation(sidecar)
        with patch.object(report, 'read', side_effect=lambda path: sidecar if path == report.RUN/filename else original(path)):
            with self.assertRaises(ValueError):
                report.checked()

    def test_complete_scope(self):
        self.assertEqual((len(self.rows), len(self.preps), len(self.outcomes)), (404, 457, 1828))
        self.assertTrue(all(row['renderedBody'] is None for row in self.rows))

    def test_missing_response_rejected(self):
        self.corrupt(lambda rows, *rest: rows.pop())

    def test_missing_skip_rejected(self):
        self.corrupt(lambda rows, manifest, review, outcomes, *rest: outcomes.pop())

    def test_false_body_credit_rejected(self):
        self.corrupt(lambda rows, *rest: rows[0].update(renderedBody='Invented body'))

    def test_false_normalized_operation_rejected(self):
        self.corrupt(lambda rows, manifest, review, *rest: review['reviews'][0].update(normalizedChoices={'e0': 'keep'}))

    def test_false_control_parity_rejected(self):
        self.corrupt_sidecar('control-parity.json', lambda value: value['records'][0].update(sameRawChoice=False))

    def test_false_label_agreement_rejected(self):
        self.corrupt_sidecar('counterbalance-comparison.json', lambda value: value['records'][0].update(normalizedChoiceAgrees=not value['records'][0]['normalizedChoiceAgrees']))

    def test_missing_counterbalance_pair_rejected(self):
        self.corrupt_sidecar('counterbalance-comparison.json', lambda value: value['records'].pop())

    def test_false_qualification_rejected(self):
        self.corrupt(lambda rows, manifest, review, outcomes, preps, decision: decision.update(qualified=True))


if __name__ == '__main__':
    unittest.main()
