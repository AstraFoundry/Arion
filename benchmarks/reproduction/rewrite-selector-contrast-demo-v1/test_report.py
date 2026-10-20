"""Reject unsupported serialization-only evidence and incomplete source coverage."""
import copy
import json
import unittest
from unittest.mock import patch
import report


class ContrastDemoEvidenceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.rows, cls.labels = report.checked()
        cls.manifest, cls.review, cls.decision = [report.read(report.RUN/name) for name in ['manifest.json', 'review.json', 'selection.json']]
        cls.outcomes = list(map(json.loads, (report.RUN/'case-outcomes.jsonl').read_text().splitlines()))
        cls.preps = list(map(json.loads, (report.RUN/'preparations.jsonl').read_text().splitlines()))

    def corrupt_core(self, operation):
        values = copy.deepcopy([self.rows, self.manifest, self.review, self.outcomes, self.preps, self.decision])
        operation(*values)
        with self.assertRaises(ValueError):
            report.core.validate(*values, report.RUN)

    def corrupt_sidecar(self, filename, operation):
        original = report.read
        sidecar = original(report.RUN/filename)
        operation(sidecar)
        with patch.object(report, 'read', side_effect=lambda path: sidecar if path == report.RUN/filename else original(path)):
            with self.assertRaises(ValueError):
                report.checked()

    def test_complete_source_and_response_scope(self):
        self.assertEqual((len(self.rows), len(self.preps), len(self.outcomes)), (404, 457, 1828))
        self.assertTrue(all(row['renderedBody'] is None for row in self.rows))

    def test_missing_actual_response_rejected(self):
        self.corrupt_core(lambda rows, *rest: rows.pop())

    def test_missing_skip_rejected(self):
        self.corrupt_core(lambda rows, manifest, review, outcomes, *rest: outcomes.pop())

    def test_final_body_credit_rejected(self):
        self.corrupt_core(lambda rows, *rest: rows[0].update(renderedBody='Invented body'))

    def test_false_control_parity_rejected(self):
        self.corrupt_sidecar('control-parity.json', lambda value: value['records'][0].update(sameRawChoice=False))

    def test_false_change_claim_rejected(self):
        self.corrupt_sidecar('contrast-demo-comparison.json', lambda value: value['records'][0].update(changedChoice=not value['records'][0]['changedChoice']))

    def test_missing_order_pair_rejected(self):
        self.corrupt_sidecar('contrast-demo-comparison.json', lambda value: value['records'].pop())

    def test_changed_source_context_rejected(self):
        def corrupt(rows, *rest):
            row = next(r for r in rows if r['arm'] == 'contrast-demo')
            row['actualContextUsed'] = {}
        self.corrupt_core(corrupt)

    def test_false_added_demo_count_rejected(self):
        self.corrupt_sidecar('contrast-demo-comparison.json', lambda value: value['records'][0].update(addedDemoMessages=0))

    def test_invented_preflight_generations_rejected(self):
        path=report.RUN.parent/'preflight/status.json';original=report.read;value=original(path);value['modelGenerations']=404
        with patch.object(report,'read',side_effect=lambda p:value if p==path else original(p)):
            with self.assertRaises(ValueError):report.checked()

    def test_false_qualification_rejected(self):
        self.corrupt_core(lambda rows, manifest, review, outcomes, preps, decision: decision.update(qualified=True))


if __name__ == '__main__':
    unittest.main()
