"""Reuse complete model-screen corruption checks and test native probe boundaries."""
import copy
import unittest
from unittest.mock import patch
import report

base = report.module('hybrid_prior_evidence_tests', report.HERE.parent/'rewrite-selector-model-cpu-v1/test_report.py')
base.report = report


class HybridEvidenceTests(base.ModelComparisonEvidenceTests):
    def corrupt_native_read(self, name, mutate):
        original = report.read
        path = report.PREFLIGHT/name
        value = copy.deepcopy(original(path))
        mutate(value)
        with patch.object(report, 'read', side_effect=lambda target: value if target == path else original(target)):
            with self.assertRaises(ValueError):
                report.preflight_checked()

    def test_wrong_cash_choice_cannot_gain_runtime_task_credit(self):
        def change(value):
            negative = next(item for item in value['reviews'] if item['id']=='99-other-source')
            negative['probeConforms'] = True
        self.corrupt_native_read('review.json', change)

    def test_false_cache_parity_is_rejected(self):
        self.corrupt_native_read('qualification.json', lambda value: value['cacheRawParity'].update({'99': False}))

    def test_native_qualification_is_not_release_acceptance(self):
        self.corrupt_native_read('qualification.json', lambda value: value.update(productionQualified=True))

    def test_wrong_actual_native_template_is_rejected(self):
        self.corrupt_native_read('hybrid-preflight-99-props.json', lambda value: value.update(chat_template='Changed template'))


if __name__ == '__main__':
    unittest.main()
