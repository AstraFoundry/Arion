"""Check independent homophone/unique controls and literal-source protection."""
import unittest
from pronunciation import Lexicon
class PronunciationTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):cls.lexicon=Lexicon()
 def test_spelling_does_not_resolve_cash_cache_homophones(self):
  result=self.lexicon.align('先查 read cash，不改接口。','readCache、readCash、readCache','work')[0];self.assertEqual(result['status'],'ambiguous');self.assertEqual(result['candidateSpellings'],['readCache','readCash'])
 def test_unique_complete_function_spelling(self):
  result=self.lexicon.align('给 load cash 加日志。','func loadCache(key string); computeValue','prompt')[0];self.assertEqual(result['status'],'unique');self.assertEqual(result['candidateSpellings'],['loadCache']);self.assertEqual('给 load cash 加日志。'.encode()[result['startUtf8']:result['endUtf8']].decode(),'load cash')
 def test_literal_and_unclosed_quotes_are_not_repair_spans(self):
  for source in ['原文“load cash”不要改。','保留`load cash`原样。','原文“load cash']:
   self.assertEqual(self.lexicon.align(source,'loadCache','work'),[])
 def test_unknown_word_is_explicit_not_a_false_unique_candidate(self):
  result=self.lexicon.align('先查 zorb quix。','zorbQuix','work')[0];self.assertEqual(result['status'],'unknown_pronunciation');self.assertEqual(result['candidateSpellings'],[])
 def test_scene_scope_and_exact_identifier(self):
  self.assertEqual(self.lexicon.align('先查 readCash。','readCache、readCash','work'),[]);self.assertEqual(self.lexicon.align('read cash','readCache','chat'),[])
if __name__=='__main__':unittest.main()
