"""Protect zero-generation failures and exact runtime-probe projections."""
import json
import pathlib
import shutil
import tempfile
import unittest
import report


class DiagnosticEvidenceTests(unittest.TestCase):
    def test_actual_execution_scope(self):
        summary, rows, _ = report.checked()
        self.assertEqual((summary['completedResponses'], summary['initialCompletedGenerations']), (7, 0))
        self.assertEqual(len(summary['errors']), 6)
        self.assertTrue(all(row['stage'] == 'runtime-probe' for row in rows))

    def mutate(self, relative, operation):
        with tempfile.TemporaryDirectory() as temp:
            folder = pathlib.Path(temp)/'evidence'
            shutil.copytree(report.RUN, folder)
            path = folder/relative
            value = json.loads(path.read_text())
            operation(value)
            path.write_text(json.dumps(value))
            with self.assertRaises(ValueError):
                report.checked(folder)

    def test_initial_failure_cannot_be_completed(self):
        self.mutate('failed-schema/manifest.json', lambda value: value.update(status='complete'))

    def test_missing_model_response_is_rejected(self):
        self.mutate('diagnostic-responses/manifest.json', lambda value: value.update(generations=6))

    def test_native_raw_source_change_is_rejected(self):
        self.mutate('precision/results.json', lambda value: value[0].update(httpError=None))

    def test_review_change_is_rejected(self):
        self.mutate('diagnostic-responses/review.json', lambda value: value['reviews'][0].update(probeConforms=True))


if __name__ == '__main__':
    unittest.main()
