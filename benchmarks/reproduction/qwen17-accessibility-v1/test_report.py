"""Model-free integrity checks for the native context report."""
import importlib.util
import json
import pathlib
import shutil
import tempfile
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[3]
SPEC = importlib.util.spec_from_file_location('context_report', pathlib.Path(__file__).with_name('report.py'))
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class ContextReportTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.run = pathlib.Path(self.temp.name) / 'run'
        shutil.copytree(ROOT / 'benchmarks/runs/qwen17-accessibility-v1/window', self.run)

    def rebind(self, filename):
        path = self.run / 'review.json'
        review = json.loads(path.read_text())
        review['files'][filename] = MODULE.sha(self.run / filename)
        path.write_text(json.dumps(review))

    def test_completed_pair(self):
        result = MODULE.report(self.run)
        self.assertEqual(result['equalFinalOutputs'], 12)
        self.assertEqual(result['pairedWindowReads'], 36)

    def test_missing_read_cannot_complete(self):
        p = self.run / 'window-observations.jsonl'
        p.write_text('\n'.join(p.read_text().splitlines()[1:]))
        self.rebind(p.name)
        with self.assertRaises(ValueError):
            MODULE.report(self.run)

    def test_unreviewed_changed_output_cannot_complete(self):
        p = self.run / 'ax-rewrite.jsonl'
        p.write_text(p.read_text().replace('林语桐', '林玉桐'))
        with self.assertRaises(ValueError):
            MODULE.report(self.run)

    def test_canary_leak_cannot_complete(self):
        p = self.run / 'window-observations.jsonl'
        data = MODULE.rows(p)
        data[0]['axText'] += 'SECRET_CANARY_NEVER_READ'
        p.write_text('\n'.join(json.dumps(r) for r in data))
        self.rebind(p.name)
        with self.assertRaises(ValueError):
            MODULE.report(self.run)

    def test_integration_uses_actual_asr_and_window_text(self):
        result = MODULE.integration_report(ROOT / 'benchmarks/runs/qwen17-accessibility-v1/integration', self.run)
        self.assertEqual((result['oldPass'], result['newPass']), (10, 8))
        self.assertEqual(len(result['regressed']), 3)

    def test_fabricated_integration_source_cannot_complete(self):
        directory = pathlib.Path(self.temp.name) / 'integration'
        shutil.copytree(ROOT / 'benchmarks/runs/qwen17-accessibility-v1/integration', directory)
        filename = directory / 'integration-new-corpus.json'
        corpus = json.loads(filename.read_text())
        corpus['samples'][0]['sourceText'] = 'invented ASR output'
        filename.write_text(json.dumps(corpus))
        review_path = directory / 'review.json'
        review = json.loads(review_path.read_text())
        review['files'][filename.name] = MODULE.sha(filename)
        review_path.write_text(json.dumps(review))
        with self.assertRaises(ValueError):
            MODULE.integration_report(directory, self.run)


if __name__ == '__main__':
    unittest.main()
