"""Verify coverage broadening retains literal/source-letter/ambiguity boundaries."""
import unittest
import pipeline
class ToneCandidateTests(unittest.TestCase):
 def test_different_tone_candidate_is_unverified(self):
  source='这份验受报告先别发。';context={'Screen':'验收报告审核'};old=pipeline.candidates_for('baseline').prepare(source,context,'chat');new=pipeline.candidates_for('candidate').prepare(source,context,'chat');self.assertEqual(old['hypotheses'],[]);self.assertEqual(new['source'],source);self.assertEqual(new['hypotheses'][0]['candidateSpellings'],['验收报告']);self.assertEqual(new['hypotheses'][0]['exactToneCandidates'],[])
 def test_candidate_is_not_automatic_semantic_truth(self):
  source='苗圃的培植清单先核对，别改成系统配置。';p=pipeline.candidates_for('candidate').prepare(source,{'Screen':'部署配置清单'},'work');self.assertEqual(p['source'],source);self.assertTrue(any('配置清单' in h['candidateSpellings'] for h in p['hypotheses']))
 def test_quoted_words_excluded(self):
  p=pipeline.candidates_for('candidate').prepare('请原样写“验受报告”。',{'Screen':'验收报告审核'},'prompt');self.assertEqual(p['hypotheses'],[])
 def test_exact_source_spelling_retained(self):
  p=pipeline.candidates_for('candidate').prepare('请核对合同。',{'Screen':'孩童活动方案；合同模板另附。'},'work');self.assertFalse(any(h['sourceSpan']=='合同' for h in p['hypotheses']))
 def test_source_letters_choose_canonical(self):
  p=pipeline.candidates_for('candidate').prepare('先查 read cash 的记录。',{'Screen':'readCash、readCache'},'work');self.assertEqual(p['hypotheses'][0]['candidateSpellings'],['readCash'])
 def test_ambiguous_alternatives_only_keep(self):
  c=pipeline.candidates_for('candidate');p=c.prepare('先查 reed cash 的记录。',{'Screen':'readCash、readCache'},'work');self.assertEqual(p['hypotheses'][0]['status'],'ambiguous');self.assertEqual(c.schema(p)['properties']['e0']['enum'],['keep'])
if __name__=='__main__':unittest.main()
