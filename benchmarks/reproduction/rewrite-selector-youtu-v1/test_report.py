"""Reject corrupted native/protocol claims and incomplete actual selector evidence."""
import copy,unittest
from unittest.mock import patch
import report
base=report.module('youtu_base_tests',report.HERE.parent/'rewrite-selector-model-cpu-v1/test_report.py');base.report=report
class YoutuEvidenceTests(base.ModelComparisonEvidenceTests):
    def corrupt(self,path,mutate,checker):
        read=report.read;value=copy.deepcopy(read(path));mutate(value)
        with patch.object(report,'read',side_effect=lambda p:value if p==path else read(p)),patch.object(report.policy,'read',side_effect=lambda p:value if p==path else read(p)):
            with self.assertRaises(ValueError):checker()
    def test_missed_positive_cannot_gain_credit(self):
        self.corrupt(report.PREFLIGHT/'review.json',lambda value:next(x for x in value['reviews'] if x['id']=='99-free-positive').update(probeConforms=True),report.preflight_checked)
    def test_cache_claim_cannot_change(self):
        self.corrupt(report.PREFLIGHT/'qualification.json',lambda value:value['cacheRawParity'].update({'99':False}),report.preflight_checked)
    def test_runtime_is_not_release(self):
        self.corrupt(report.PREFLIGHT/'qualification.json',lambda value:value.update(productionQualified=True),report.preflight_checked)
    def test_actual_template_cannot_change(self):
        self.corrupt(report.PREFLIGHT/'hybrid-preflight-99-props.json',lambda value:value.update(chat_template='Wrong template'),report.preflight_checked)
    def test_identity_template_cannot_gain_final_lf(self):
        self.corrupt(report.policy.template_path,lambda value:value['metadata'].update({'tokenizer.chat_template':value['metadata']['tokenizer.chat_template']+'\n'}),report.preflight_checked)
if __name__=='__main__':unittest.main()
