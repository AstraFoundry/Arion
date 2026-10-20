"""Exercise source coverage, quoted splitting and anchor construction."""
import unittest
import plan
class PlanTests(unittest.TestCase):
 def test_quotes_are_not_split(self):self.assertEqual(plan.units('把“甲，乙”保留，别改。','work'),['把“甲，乙”保留，','别改。'])
 def test_chat_is_one_unit(self):self.assertEqual(plan.units('如果有雨，就别去。','chat'),['如果有雨，就别去。'])
 def test_ambiguous_anchor_is_locked(self):
  data=plan.prepare(dict(source='先查 read cash，别改。',scene='work',context={'Screen':'readCache readCash'}));self.assertEqual(data['units'][0]['anchors'][0]['allowedSpellings'],['read cash'])
 def test_novel_identifier_can_be_selected(self):
  data=plan.prepare(dict(source='查 zorb quix。',scene='prompt',context={'Screen':'zorbQuix'}));self.assertEqual(data['units'][0]['anchors'][0]['allowedSpellings'],['zorb quix','zorbQuix'])
if __name__=='__main__':unittest.main()
