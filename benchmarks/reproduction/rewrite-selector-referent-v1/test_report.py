"""Reject missing public-description evidence and unsupported joint eligibility."""
import copy
import json
import unittest
import report


class ReferentEvidenceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.rows, cls.labels = report.checked()
        cls.manifest = report.read(report.RUN/'manifest.json')
        cls.review = report.read(report.RUN/'review.json')
        cls.outcomes = list(map(json.loads, (report.RUN/'case-outcomes.jsonl').read_text().splitlines()))
        cls.preps = list(map(json.loads, (report.RUN/'preparations.jsonl').read_text().splitlines()))
        cls.decision = report.read(report.RUN/'selection.json')

    def corrupt(self, operation):
        values = copy.deepcopy([self.rows, self.manifest, self.review, self.outcomes, self.preps, self.decision])
        operation(*values)
        with self.assertRaises(ValueError):
            report.validate(*values)

    def test_complete_scope(self):
        self.assertEqual(len(self.rows), 404)
        self.assertEqual(len(self.preps), 457)
        self.assertEqual(len(self.outcomes), 1828)
        self.assertTrue(all(row['renderedBody'] is None for row in self.rows))

    def test_missing_actual_choice_is_rejected(self):
        self.corrupt(lambda rows, *rest: rows.pop())

    def test_missing_no_hypothesis_outcome_is_rejected(self):
        self.corrupt(lambda rows, manifest, review, outcomes, *rest: outcomes.pop())

    def test_false_final_body_is_rejected(self):
        self.corrupt(lambda rows, *rest: rows[0].update(renderedBody='Invented final body'))

    def test_control_cannot_gain_description_credit(self):
        self.corrupt(lambda rows, manifest, review, *rest: review['reviews'][0].update(descriptionConforms=True))

    def test_normalized_choice_is_bound_to_raw(self):
        self.corrupt(lambda rows, manifest, review, *rest: review['reviews'][0].update(normalizedChoices={'e0': 'keep'}))

    def test_wrong_description_text_is_rejected(self):
        def change(rows, manifest, review, *rest):
            item = next(j for j in review['reviews'] if j['descriptionReview'] is not None)
            next(iter(item['descriptionReview'].values()))['text'] = 'Changed description'
        self.corrupt(change)

    def test_joint_eligibility_cannot_ignore_failures(self):
        self.corrupt(lambda rows, manifest, review, outcomes, preps, decision: decision.update(eligibleArms=['referent']))


if __name__ == '__main__':
    unittest.main()
