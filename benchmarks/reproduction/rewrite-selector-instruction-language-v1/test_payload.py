"""Ensure language translation never changes data, examples, candidates or decoding."""
import json
import pathlib
import unittest
import pipeline


class InstructionLanguageTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.root = pathlib.Path(__file__).resolve().parents[3]
        cls.preps = list(map(json.loads, (cls.root/'benchmarks/runs/rewrite-selector-screen-v1/run/preparations.jsonl').read_text().splitlines()))

    def test_complete_source_projection(self):
        for prep in self.preps:
            for arm in pipeline.ARMS:
                request, prepared = pipeline.selection_payload(arm, prep['sample'], prep['cleanupObservation']['cleanup'])
                expected, original = pipeline.prior.selection_payload(arm, prep['sample'], prep['cleanupObservation']['cleanup'])
                self.assertEqual(prepared, original)
                self.assertEqual(prepared, prep['prepared'])
                self.assertEqual(pipeline.chinese_projection(request, arm), expected)

    def test_clause_translation_preserves_every_chinese_rule(self):
        prompt = json.loads((pipeline.HERE/'prompts.json').read_text())
        original = json.loads((pipeline.HERE.parent/'rewrite-selector-screen-v1/templates/baseline/prompts.json').read_text())['selectionSystem']
        self.assertEqual(''.join(item['chinese'] for item in prompt['clauseAlignment']), original)
        self.assertEqual(' '.join(item['english'] for item in prompt['clauseAlignment']), prompt['selectionSystem'])

    def test_only_system_message_changes(self):
        prep = next(p for p in self.preps if p['prepared']['hypotheses'])
        for arm in pipeline.ARMS:
            actual, _ = pipeline.selection_payload(arm, prep['sample'], prep['cleanupObservation']['cleanup'])
            prior, _ = pipeline.prior.selection_payload(arm, prep['sample'], prep['cleanupObservation']['cleanup'])
            self.assertNotEqual(actual['messages'][0], prior['messages'][0])
            self.assertEqual(actual['messages'][1:], prior['messages'][1:])
            self.assertEqual({k:v for k,v in actual.items() if k!='messages'}, {k:v for k,v in prior.items() if k!='messages'})

    def test_complete_hostile_context_stays_data(self):
        prep = next(p for p in self.preps if p['sample']['id']=='history-0277')
        request, _ = pipeline.selection_payload('balanced', prep['sample'], prep['cleanupObservation']['cleanup'])
        self.assertEqual(json.loads(request['messages'][-1]['content'])['context'], prep['sample']['context'])
        self.assertIn('DONE', json.loads(request['messages'][-1]['content'])['context']['Screen'])

    def test_no_evaluator_fields_enter_request(self):
        prep = self.preps[0]
        request, _ = pipeline.selection_payload('baseline', prep['sample'], prep['cleanupObservation']['cleanup'])
        data = json.loads(request['messages'][-1]['content'])
        self.assertNotIn('allowedChoices', data)
        self.assertNotIn('requiredRepairs', data)

    def test_unknown_arm_rejected(self):
        with self.assertRaises(ValueError):
            pipeline.selection_payload('unknown', self.preps[0]['sample'], self.preps[0]['cleanupObservation']['cleanup'])


if __name__ == '__main__':
    unittest.main()
