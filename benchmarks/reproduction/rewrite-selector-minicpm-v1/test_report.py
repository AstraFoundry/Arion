"""Reuse complete comparison corruption checks and protect native qualification scope."""
import copy
import unittest
from unittest.mock import patch
import report
base=report.module('minicpm_base_evidence_tests',report.HERE.parent/'rewrite-selector-model-cpu-v1/test_report.py')
base.report=report
class MiniCpmEvidenceTests(base.ModelComparisonEvidenceTests):
    def corrupt(self,path,mutate):
        original=report.read;value=copy.deepcopy(original(path));mutate(value)
        with patch.object(report,'read',side_effect=lambda target:value if target==path else original(target)):
            with self.assertRaises(ValueError):report.preflight_checked()
    def test_literal_cash_failure_cannot_gain_credit(self):
        self.corrupt(report.PREFLIGHT/'review.json',lambda value:next(j for j in value['reviews'] if j['id']=='99-other-source').update(probeConforms=True))
    def test_missed_positive_failure_cannot_gain_credit(self):
        self.corrupt(report.PREFLIGHT/'review.json',lambda value:next(j for j in value['reviews'] if j['id']=='99-cache-off-first').update(probeConforms=True))
    def test_cache_parity_cannot_be_invented(self):
        self.corrupt(report.PREFLIGHT/'qualification.json',lambda value:value['cacheRawParity'].update({'99':False}))
    def test_runtime_is_not_release_acceptance(self):
        self.corrupt(report.PREFLIGHT/'qualification.json',lambda value:value.update(productionQualified=True))
    def test_actual_template_cannot_change(self):
        self.corrupt(report.PREFLIGHT/'hybrid-preflight-99-props.json',lambda value:value.update(chat_template='Wrong template'))
    def test_derived_native_template_must_remove_exactly_one_lf(self):
        self.corrupt(report.policy.template_path,lambda value:value['metadata'].update({'tokenizer.chat_template':value['metadata']['tokenizer.chat_template']+'\n'}))
    def test_aborted_native_completion_cannot_be_hidden(self):
        self.corrupt(report.PREFLIGHT.parent/'aborted-before-qualification/closure.json',lambda value:value.update(nativeLogCompletions=0))
if __name__=='__main__':unittest.main()
