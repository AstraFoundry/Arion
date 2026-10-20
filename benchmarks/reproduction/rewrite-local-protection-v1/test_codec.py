"""Check reversible offsets, literal spans and explicit protocol failure coverage."""
import unittest
import codec
class CodecTests(unittest.TestCase):
 def sample(self,text,screen):return dict(source=text,context={'Screen':screen},scene='work')
 def test_utf8_and_multiple_runs(self):
  sample=self.sample('先查 read cash，再查 write right。','readCache readCash writeRight writeRite');encoded,records,_=codec.encode(sample);self.assertEqual(len(records),2);self.assertEqual(codec.decode(encoded,dict(protectedTerms=records)),sample['source'])
 def test_unaffected_requests_are_identical(self):
  sample=self.sample('先别提交。','readCash');template=dict(system='Edit only',examples=[],includePronunciationEvidence=True);self.assertEqual(codec.messages(sample,template,True)[0],codec.messages(sample,template,False)[0])
 def test_unique_is_editable(self):self.assertEqual(codec.encode(self.sample('给 push cash 加日志。','pushCache'))[1],[])
 def test_quotes_and_exact_remain_literal(self):self.assertEqual(codec.encode(self.sample('保留“read cash”和 readCache。','readCache readCash'))[1],[])
 def test_missing_marker_fails(self):
  _,records,_=codec.encode(self.sample('查 read cash。','readCache readCash'))
  with self.assertRaises(ValueError):codec.decode('查 readCache。',dict(protectedTerms=records))
 def test_duplicate_marker_fails(self):
  encoded,records,_=codec.encode(self.sample('查 read cash。','readCache readCash'))
  with self.assertRaises(ValueError):codec.decode(encoded+records[0]['token'],dict(protectedTerms=records))
 def test_preserves_other_generated_content(self):
  encoded,records,_=codec.encode(self.sample('查 read cash。','readCache readCash'));self.assertIn('无依据的新内容',codec.decode(encoded+'无依据的新内容',dict(protectedTerms=records)))
if __name__=='__main__':unittest.main()
