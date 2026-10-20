"""Protect full CPU selector coverage, exact paired requests and absent delivery claims."""
import copy
import json
import unittest
import report


class ModelComparisonEvidenceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.rows, cls.labels, cls.targets = report.checked()
        cls.manifest = report.read(report.RUN/'manifest.json')
        cls.review = report.read(report.RUN/'review.json')
        cls.outcomes = list(map(json.loads, (report.RUN/'case-outcomes.jsonl').read_text().splitlines()))
        cls.decision = report.read(report.RUN/'selection.json')

    def check(self, operation):
        values = copy.deepcopy([self.rows, self.manifest, self.review, self.outcomes, self.decision])
        operation(*values)
        with self.assertRaises(ValueError):
            report.validate(*values)

    def test_actual_scope(self):
        self.assertEqual(len(self.rows), 202)
        self.assertEqual(len(self.outcomes), 914)
        self.assertTrue(all(row['renderedBody'] is None for row in self.rows))

    def test_missing_selector_is_rejected(self):
        self.check(lambda rows, *rest: rows.pop())

    def test_missing_skip_is_rejected(self):
        self.check(lambda rows, manifest, review, outcomes, decision: outcomes.pop())

    def test_invented_final_body_is_rejected(self):
        self.check(lambda rows, *rest: rows[0].update(renderedBody='Invented body'))

    def test_gold_added_to_actual_request_is_rejected(self):
        self.check(lambda rows, *rest: rows[0]['request']['messages'].append(dict(role='user', content='Gold reference')))

    def test_source_context_change_is_rejected(self):
        self.check(lambda rows, *rest: rows[0].update(context={}))

    def test_false_model_adoption_is_rejected(self):
        self.check(lambda rows, manifest, review, outcomes, decision: decision.update(qualified=True))

    def test_new_cleanup_claim_is_rejected(self):
        self.check(lambda rows, manifest, *rest: manifest.update(newGoPreparations=457))


if __name__ == '__main__':
    unittest.main()
