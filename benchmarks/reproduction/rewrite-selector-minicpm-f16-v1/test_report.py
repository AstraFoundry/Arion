"""Protect full F16 coverage, native facts and exact Q4 comparison bindings."""
import copy,unittest
from unittest.mock import patch
import report
base=report.module('f16_base_model_tests',report.HERE.parent/'rewrite-selector-model-cpu-v1/test_report.py');base.report=report
class F16EvidenceTests(base.ModelComparisonEvidenceTests):
    def corrupt(self,path,mutate,checker):
        read=report.read;value=copy.deepcopy(read(path));mutate(value)
        # The native helper receives the original read callable through its explicit policy.
        with patch.object(report,'read',side_effect=lambda p:value if p==path else read(p)),patch.object(report.policy,'read',side_effect=lambda p:value if p==path else read(p)):
            with self.assertRaises(ValueError):checker()
    def test_literal_cash_error_cannot_gain_credit(self):
        self.corrupt(report.PREFLIGHT/'review.json',lambda value:next(x for x in value['reviews'] if x['id']=='99-other-source').update(probeConforms=True),report.preflight_checked)
    def test_cache_claim_cannot_change(self):
        self.corrupt(report.PREFLIGHT/'qualification.json',lambda value:value['cacheRawParity'].update({'99':False}),report.preflight_checked)
    def test_runtime_is_not_release(self):
        self.corrupt(report.PREFLIGHT/'qualification.json',lambda value:value.update(productionQualified=True),report.preflight_checked)
    def test_actual_template_cannot_change(self):
        self.corrupt(report.PREFLIGHT/'hybrid-preflight-99-props.json',lambda value:value.update(chat_template='Wrong template'),report.preflight_checked)
    def test_exact_native_final_lf_handling(self):
        self.corrupt(report.policy.template_path,lambda value:value['metadata'].update({'tokenizer.chat_template':value['metadata']['tokenizer.chat_template']+'\n'}),report.preflight_checked)
    def test_q4_comparison_cannot_invent_a_change(self):
        self.corrupt(report.RUN/'q4-comparison.json',lambda value:value['records'][0].update(changedChoice=not value['records'][0]['changedChoice']),report.checked)
    def test_q4_comparison_cannot_drop_a_pair(self):
        self.corrupt(report.RUN/'q4-comparison.json',lambda value:value['records'].pop(),report.checked)
if __name__=='__main__':unittest.main()
