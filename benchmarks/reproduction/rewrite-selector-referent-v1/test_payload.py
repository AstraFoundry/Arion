"""Check explanatory payload isolation and exact bounded candidate application."""
import json
import unittest
import pipeline


class ReferentPayloadTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.sample = dict(id='payload-probe', source='给 fetch cash 加日志，不改设置。', context={'Screen': 'cache.go: func fetchCache(key string)'}, scene='work')
        cls.cleanup = dict(text=cls.sample['source'], version='speech-spans-v2')
        cls.requests = {arm: pipeline.selection_payload(arm, cls.sample, cls.cleanup) for arm in pipeline.ARMS}

    def test_controls_only_change_budget(self):
        for arm in ['baseline', 'balanced']:
            old, prepared = pipeline.prior.selection_payload(arm, self.sample, self.cleanup)
            new, actual = self.requests[arm]
            old['max_tokens'] = 256
            self.assertEqual(new, old)
            self.assertEqual(actual, prepared)

    def test_all_inputs_project_to_exact_control(self):
        request = self.requests['baseline'][0]
        for arm, (candidate, _) in self.requests.items():
            self.assertEqual(pipeline.baseline_projection(candidate, arm), request)

    def test_hypotheses_are_identical_across_arms(self):
        self.assertTrue(all(prepared == self.requests['baseline'][1] for _, prepared in self.requests.values()))

    def test_public_description_is_not_delivered_or_applied(self):
        prepared = self.requests['referent'][1]
        raw = json.dumps({'e0': dict(referent='Untrusted description adds a deadline tomorrow.', choice='replace')})
        source, edits = pipeline.candidates_for('referent').apply(raw, prepared)
        self.assertEqual(source, '给 fetchCache 加日志，不改设置。')
        self.assertEqual(len(edits), 1)
        self.assertNotIn('tomorrow', source)

    def test_unknown_ids_are_rejected(self):
        prepared = self.requests['referent'][1]
        with self.assertRaises(ValueError):
            pipeline.candidates_for('referent').apply(json.dumps({'e8': dict(referent='Unknown', choice='replace')}), prepared)

    def test_missing_description_is_rejected(self):
        with self.assertRaises(ValueError):
            pipeline.candidates_for('referent').apply('{"e0":{"choice":"keep"}}', self.requests['referent'][1])

    def test_invalid_choice_still_uses_original_protocol(self):
        with self.assertRaises(ValueError):
            pipeline.candidates_for('referent').apply('{"e0":{"referent":"Same function","choice":"invent"}}', self.requests['referent'][1])

    def test_boolean_description_is_rejected(self):
        with self.assertRaises(ValueError):
            pipeline.candidates_for('referent').apply('{"e0":{"referent":true,"choice":"keep"}}', self.requests['referent'][1])


if __name__ == '__main__':
    unittest.main()
