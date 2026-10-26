"""Model-free integrity and metric oracles; these tests are not inference evidence."""
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import study


class ReportTests(unittest.TestCase):
    def test_candidate_identity_rejects_fabrication_and_duplicates(self):
        for identities, ranked in [([1, 1], ["a", "a"]), ([1, 3], ["a", "c"]), ([2, 1], ["wrong", "a"])]:
            with self.assertRaises(ValueError):
                study.validate_row({"source": ["a", "b"], "identities": identities, "ranked": ranked})

    def test_type7_percentile(self):
        self.assertEqual(study.percentile([0, 10], .95), 9.5)
        self.assertIsNone(study.percentile([], .95))

    def fixture(self, directory):
        root = Path(directory)
        (root / "requests").mkdir()
        (root / "raw").mkdir()
        corpus = root / "corpus.json"
        study.save(corpus, {"cases": [{"id": "rescue", "group": "x", "acceptable": ["b"]},
                                     {"id": "oracle-miss", "group": "x", "acceptable": ["z"]}]})
        study.save(root / "status.json", {"status": "completed", "cases": 2})
        study.save(root / "metadata.json", {"mode": "final", "corpusSha256": study.digest(corpus)})
        index = []
        for name in ["rescue", "oracle-miss"]:
            row = {"id": "key-1", "source": ["a", "b"], "ranked": ["b", "a"], "identities": [2, 1],
                   "preedit": "ab", "status": "completed", "wallMs": 10, "processCpuMs": 1, "requests": []}
            request = {"id": name, "config": {}, "history": "", "steps": [{"id": "key-1", "preedit": "ab", "candidates": ["a", "b"]}]}
            study.save(root / "requests" / (name + ".json"), request)
            study.save(root / "raw" / (name + ".json"), {"id": name, "steps": [row]})
            index.append({"id": name, "requestSha256": study.digest(root / "requests" / (name + ".json")), "responseSha256": study.digest(root / "raw" / (name + ".json"))})
        study.save(root / "observations.json", index)
        return root, corpus

    def test_rescue_and_oracle_miss_denominator(self):
        with tempfile.TemporaryDirectory() as directory:
            root, corpus = self.fixture(directory)
            with patch.object(study, "CORPUS", corpus):
                result = study.evaluate(root)
            self.assertEqual(result["rerankedTop1"], .5)
            self.assertEqual(result["baselineMRR9"], .25)
            self.assertEqual(result["rerankedMRR9"], .5)
            self.assertEqual(result["counts"]["rescues"], 1)
            self.assertEqual(result["counts"]["oracle9"], 1)

    def test_changed_result_and_incomplete_run_fail(self):
        with tempfile.TemporaryDirectory() as directory:
            root, corpus = self.fixture(directory)
            with patch.object(study, "CORPUS", corpus):
                file = root / "raw/rescue.json"
                file.write_text(file.read_text() + " ")
                with self.assertRaisesRegex(ValueError, "hash mismatch"):
                    study.evaluate(root)
                study.save(root / "status.json", {"status": "failed"})
                with self.assertRaisesRegex(ValueError, "incomplete"):
                    study.evaluate(root)

    def test_duplicate_and_label_leak_fail_even_with_updated_hash(self):
        with tempfile.TemporaryDirectory() as directory:
            root, corpus = self.fixture(directory)
            with patch.object(study, "CORPUS", corpus):
                entries = study.read(root / "observations.json")
                study.save(root / "observations.json", [entries[0], entries[0]])
                with self.assertRaisesRegex(ValueError, "duplicate"):
                    study.evaluate(root)
                request_file = root / "requests/rescue.json"
                request = study.read(request_file)
                request["acceptable"] = ["b"]
                study.save(request_file, request)
                entries[0]["requestSha256"] = study.digest(request_file)
                study.save(root / "observations.json", entries)
                with self.assertRaisesRegex(ValueError, "label leak"):
                    study.evaluate(root)


if __name__ == "__main__":
    unittest.main()
