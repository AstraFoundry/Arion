"""Exercise report completeness and source/output binding without inference."""
import copy
import hashlib
import json
import pathlib
import shutil
import tempfile
import unittest
import report

class ContextReportIntegrity(unittest.TestCase):
    def setUp(self):
        self.temporary=tempfile.TemporaryDirectory();self.addCleanup(self.temporary.cleanup)
        self.directory=pathlib.Path(self.temporary.name)/'run'
        shutil.copytree(report.ROOT/'benchmarks/runs/context-repair-v1/development',self.directory)

    def review(self,change):
        p=self.directory/'review.json';data=report.read(p);change(data);p.write_text(json.dumps(data))

    def test_complete(self):
        rows,reviews=report.validate(self.directory,'context-repair-v1')
        self.assertEqual(len(rows),72);self.assertEqual(len(reviews),72)

    def test_missing_review(self):
        self.review(lambda data:data['reviews'].pop())
        with self.assertRaisesRegex(ValueError,'Missing review'):report.validate(self.directory,'context-repair-v1')

    def test_duplicate_review(self):
        self.review(lambda data:data['reviews'].append(data['reviews'][0]))
        with self.assertRaisesRegex(ValueError,'Duplicate'):report.validate(self.directory,'context-repair-v1')

    def test_reviewed_output_changed(self):
        self.review(lambda data:data['reviews'][0].update(rowSha256='0'*64))
        with self.assertRaisesRegex(ValueError,'Changed reviewed output'):report.validate(self.directory,'context-repair-v1')

    def test_raw_changed(self):
        (self.directory/'results.jsonl').write_text('{}\n')
        with self.assertRaisesRegex(ValueError,'Changed results'):report.validate(self.directory,'context-repair-v1')

    def test_generation_missing_even_with_updated_hash(self):
        path=self.directory/'results.jsonl';data='\n'.join(path.read_text().splitlines()[1:])+'\n';path.write_text(data)
        p=self.directory/'manifest.json';manifest=report.read(p);manifest['resultsSha256']=hashlib.sha256(data.encode()).hexdigest();p.write_text(json.dumps(manifest))
        with self.assertRaisesRegex(ValueError,'Missing or duplicate generation'):report.validate(self.directory,'context-repair-v1')

class SceneReportProvenance(unittest.TestCase):
    def test_production_work_mapping_is_visible(self):
        scenes=report.read(report.ROOT/'benchmarks/runs/context-repair-v1/report/scenes.json')
        item=next(x for x in scenes if x['experiment']=='context' and x['variant']=='production' and x['id']=='identifier-wrong-relevant')
        self.assertEqual(item['assignedScene'],'work')
        self.assertEqual(item['effectiveModelScene'],'chat')
        self.assertEqual(item['templateVersion'],'context-v2+live-v2')
        self.assertEqual(item['assignmentSource'],'fixture-metadata-not-automatic-detection')

    def test_structural_scene_reconstruction_is_labeled(self):
        scenes=report.read(report.ROOT/'benchmarks/runs/context-repair-v1/report/scenes.json')
        items=[x for x in scenes if x['experiment']=='structure']
        self.assertEqual(len(items),16)
        self.assertTrue(all(x['modelSceneEvidence']=='reconstructed-from-frozen-runner-direct-scene-argument' for x in items))

    def test_missing_effective_scene_is_not_silently_filled(self):
        with self.assertRaisesRegex(ValueError,'Missing effective'):
            report.scene_metadata(dict(scene='work',response=dict(template='test')))

    def test_every_case_renders_scene_metadata(self):
        for name,count in [('context-repair-v1',88),('scene-rewrite-prompt-v2',144)]:
            directory=report.ROOT/'benchmarks/runs'/name/'report'
            scenes=report.read(directory/'scenes.json')
            markdown=(directory/'review.md').read_text()
            self.assertEqual(len(scenes),count)
            self.assertEqual(markdown.count('**Scene:**'),count)
            self.assertEqual(markdown.count('**Prompt template:**'),count)
            self.assertTrue(all(x['assignmentSource']=='fixture-metadata-not-automatic-detection' for x in scenes))

if __name__=='__main__':unittest.main()
