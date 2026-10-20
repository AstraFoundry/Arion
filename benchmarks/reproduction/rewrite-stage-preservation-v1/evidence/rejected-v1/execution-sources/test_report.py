"""Fabricated fixtures test reporter arithmetic/integrity, never model quality."""
import copy
import json
import tempfile
import unittest
from pathlib import Path

import report


class ReporterTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.path = Path(self.tmp.name)
        self.cases = self.path / 'cases.json'
        self.cases.write_text(json.dumps([{'id': 'fixture', 'preview': 'test source', 'final': 'test source expanded',
            'scene': 'chat', 'screen': 'InventedName', 'pairKind': 'fabricated-reporter-test',
            'checks': ['InventedName', 'do not deploy']}]))
        self.observations = [self.observation('preview', 'InventedName'), self.observation('final', 'DifferentName')]

    def observation(self, phase, output):
        return {'kind': 'observation', 'id': 'fixture', 'phase': phase, 'mode': 'fabricated-test-only',
            'result': {'text': output}, 'request': {'Text': 'test source'}, 'elapsedMs': 10,
            'error': '', 'wire': [{'request': {'messages': [{'role': 'user', 'content': json.dumps({'source': 'InventedName'})}]},
                 'response': {'choices': [{'message': {'content': output}, 'finish_reason': 'stop'}]},
                 'status': 200, 'elapsedMs': 5}]}

    def arm(self, observations=None, ending=True):
        path = self.path / 'arm.jsonl'
        rows = [{'kind': 'start'}] + (self.observations if observations is None else observations)
        if ending:
            rows.append({'kind': 'complete'})
        path.write_text(''.join(json.dumps(row) + '\n' for row in rows))
        return report.load_arm(path, ['fixture'])

    def render(self, arm=None):
        arms = {'baseline': self.arm() if arm is None else arm}
        review = report.review_template(self.cases, arms)
        return report.build_report(self.cases, arms, review)

    def test_proxy_does_not_create_semantic_pass(self):
        output = self.render()
        summary = output['arms']['baseline']
        self.assertEqual(summary['previewFinalRegressionProxy'], 1)
        self.assertEqual(summary['selectedTermDamageProxy'], 1)
        self.assertEqual(summary['phaseDenominator'], 2)
        self.assertEqual(summary['manualReviewCounts']['unreviewed'], 2)
        self.assertEqual(summary['manualReviewCounts']['correct'], 0)
        self.assertEqual(summary['latencyMs']['formatter'], {'count': 2, 'p50': 5, 'p95': 5})

    def test_raw_formatter_damage_and_delivered_repair_are_distinct(self):
        self.observations[1]['result']['text'] = 'InventedName'
        output = self.render()
        self.assertEqual(output['arms']['baseline']['selectedTermDamageProxy'], 1)
        self.assertEqual(output['arms']['baseline']['previewFinalRegressionProxy'], 0)

    def test_failure_stays_in_denominator_without_formatter_damage_claim(self):
        self.observations[1]['error'] = 'actual runtime error'
        self.observations[1]['result']['text'] = ''
        self.observations[1]['wire'][0]['response']['choices'][0]['finish_reason'] = 'length'
        output = self.render()
        self.assertEqual(output['arms']['baseline']['phaseDenominator'], 2)
        self.assertEqual(output['arms']['baseline']['failedPhases'], 1)
        self.assertEqual(output['arms']['baseline']['selectedTermDamageProxy'], 0)

    def test_incomplete_duplicate_and_missing_phases_rejected(self):
        for rows, ending in [(self.observations, False), (self.observations * 2, True), (self.observations[:1], True)]:
            with self.subTest(rows=len(rows), ending=ending), self.assertRaises(ValueError):
                self.arm(rows, ending)

    def test_review_binds_exact_arm_and_observation_bytes(self):
        arms = {'baseline': self.arm()}
        review = report.review_template(self.cases, arms)
        for key in ('corpusSha256', 'arms'):
            changed = copy.deepcopy(review)
            changed[key] = 'tampered'
            with self.assertRaises(ValueError):
                report.build_report(self.cases, arms, changed)
        review['records'][0]['observationSha256'] = 'tampered'
        with self.assertRaises(ValueError):
            report.build_report(self.cases, arms, review)

    def test_review_requires_explicit_rationale(self):
        arms = {'baseline': self.arm()}
        review = report.review_template(self.cases, arms)
        review['records'][0]['fidelity'] = 'correct'
        with self.assertRaises(ValueError):
            report.build_report(self.cases, arms, review)
        review['records'][0]['note'] = 'Fabricated reporter test rationale, not model evidence.'
        result = report.build_report(self.cases, arms, review)
        self.assertEqual(result['arms']['baseline']['manualReviewCounts']['correct'], 1)

    def test_archive_verification_rejects_tamper(self):
        path = self.path / 'a.txt'
        path.write_bytes(b'test')
        (self.path / 'manifest.json').write_text(json.dumps({'files': {'a.txt': report.digest(b'test')}}))
        # An unlisted file is rejected as well as a modified listed file.
        with self.assertRaises(ValueError):
            report.verify_archive(self.path)
        self.cases.unlink()
        report.verify_archive(self.path)
        path.write_bytes(b'tampered')
        with self.assertRaises(ValueError):
            report.verify_archive(self.path)

    def test_gap_formatter_is_not_selector_and_fixed_terms_are_stitched(self):
        observation = self.observations[0]
        call = observation['wire'][0]
        call['request']['response_format'] = {'type': 'json_object'}
        call['request']['messages'][-1]['content'] = json.dumps({'source': 'Check InventedName.',
            'parts': [{'gap': 'g0', 'text': 'Check '}, {'text': 'InventedName', 'fixed': True}, {'gap': 'g1', 'text': '.'}]})
        call['response']['choices'][0]['message']['content'] = '{"g0":"Check ","g1":"."}'
        stage = report.stage_details(observation)[0]
        self.assertEqual(stage['kind'], 'formatter')
        self.assertEqual(stage['protocol'], 'fixed-gaps')
        self.assertEqual(stage['renderedProposal'], 'Check InventedName.')
        rendered = self.render()
        self.assertFalse(rendered['cases'][0]['arms']['baseline']['phases']['preview']['selectedTermDamageProxy'])
        call['response']['choices'][0]['message']['content'] = '{"g0":"Check ","g1":"InventedName."}'
        self.assertEqual(report.stage_details(observation)[0]['renderedProposal'], 'Check InventedNameInventedName.')
        self.assertTrue(self.render()['cases'][0]['arms']['baseline']['phases']['preview']['selectedTermDamageProxy'])

    def test_quantiles_use_nearest_rank(self):
        self.assertEqual(report.quantiles([100, 1, 2, 3]), {'count': 4, 'p50': 2, 'p95': 100})
        self.assertEqual(report.quantiles([]), {'count': 0, 'p50': None, 'p95': None})


if __name__ == '__main__':
    unittest.main()
