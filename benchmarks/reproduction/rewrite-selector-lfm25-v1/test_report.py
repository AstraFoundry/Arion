"""Reject corrupted native/protocol claims and incomplete actual selector evidence."""
import copy,unittest,pathlib
from unittest.mock import patch
import report
base=report.module('lfm25_base_tests',report.HERE.parent/'rewrite-selector-model-cpu-v1/test_report.py');base.report=report
class Lfm25EvidenceTests(base.ModelComparisonEvidenceTests):
    def corrupt(self,path,mutate,checker):
        read=report.read;value=copy.deepcopy(read(path));mutate(value)
        with patch.object(report,'read',side_effect=lambda p:value if p==path else read(p)),patch.object(report.policy,'read',side_effect=lambda p:value if p==path else read(p)):
            with self.assertRaises(ValueError):checker()
    def test_harmful_literal_cannot_gain_credit(self):
        self.corrupt(report.PREFLIGHT/'review.json',lambda value:next(x for x in value['reviews'] if x['id']=='99-free-negative').update(probeConforms=True),report.preflight_checked)
    def test_cache_claim_cannot_change(self):
        self.corrupt(report.PREFLIGHT/'qualification.json',lambda value:value['cacheRawParity'].update({'99':False}),report.preflight_checked)
    def test_runtime_is_not_release(self):
        self.corrupt(report.PREFLIGHT/'qualification.json',lambda value:value.update(productionQualified=True),report.preflight_checked)
    def test_actual_template_cannot_change(self):
        self.corrupt(report.PREFLIGHT/'hybrid-preflight-99-props.json',lambda value:value.update(chat_template='Wrong template'),report.preflight_checked)
    def test_identity_template_cannot_gain_final_lf(self):
        self.corrupt(report.policy.template_path,lambda value:value['metadata'].update({'tokenizer.chat_template':value['metadata']['tokenizer.chat_template']+'\n'}),report.preflight_checked)
    def test_wrong_layer_marker_in_binary_log_is_rejected(self):
        path=report.PREFLIGHT/'hybrid-preflight-99-server.log'
        original=pathlib.Path.read_bytes
        changed=original(path).replace(b'offloaded 17/17 layers to GPU',b'offloaded 18/18 layers to GPU')
        with patch.object(pathlib.Path,'read_bytes',lambda p:changed if p==path else original(p)):
            with self.assertRaises(ValueError):report.preflight_checked()
    def test_non_utf8_preview_does_not_hide_valid_native_markers(self):
        raw=(report.PREFLIGHT/'hybrid-preflight-99-server.log').read_bytes()
        with self.assertRaises(UnicodeDecodeError):raw.decode('utf8')
        self.assertEqual(len(report.preflight_checked()[0]),12)
    def test_qualification_note_matches_plain_assistant_prefix(self):
        self.assertNotIn('<think>',report.policy.generation_prompt_log)
        self.assertIn('plain nonthinking assistant prefix',report.read(report.PREFLIGHT/'qualification.json')['reason'])
if __name__=='__main__':unittest.main()
