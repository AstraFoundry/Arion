"""Upstream output-contract checks, not model efficacy tests."""
import unittest
from correct import detokenize


class DetokenizationTest(unittest.TestCase):
    def test_bart_matches_upstream_default(self):
        self.assertEqual(detokenize('今 天 检 查 邮 件 。', 'bart4csc'), '今天检查邮件。')

    def test_english_keeps_word_boundaries(self):
        self.assertEqual(detokenize('keep the original document', 'flanec'), 'keep the original document')

    def test_empty_is_not_replaced(self):
        self.assertEqual(detokenize('', 'bart4csc'), '')


if __name__ == '__main__': unittest.main()
