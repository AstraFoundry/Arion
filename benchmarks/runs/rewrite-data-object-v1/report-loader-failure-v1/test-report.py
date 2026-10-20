"""Validate complete reviewed evidence and safe standalone HTML embedding."""
import json
import unittest
import sys
import types
import report
class ReportTests(unittest.TestCase):
    def test_complete_evidence(self):
        data=report.collect()
        self.assertEqual(len(data['cases']),1784)
        self.assertEqual(data['naturalAsrPreparation']['sources'],120)
        self.assertFalse(data['naturalAsrPreparation']['newInference'])
        self.assertEqual(data['timingIntegrity'],dict(observations=396,flagged=9))
        self.assertEqual(len(data['rounds']),28)
        self.assertEqual(sum(c['actualMessages'] is not None for c in data['cases']),300)
        self.assertEqual(sum(c['modelSource'] is not None for c in data['cases']),180)
        self.assertEqual(sum(bool(c['reviewQualifier']) for c in data['cases']),1)
        self.assertEqual(data['naturalTextReview']['reviewed'],120)
        self.assertEqual(len(data['audits']),113)
        self.assertEqual(len(data['ocrFixtures']),12)
        self.assertEqual(sum(len(f['observations']) for f in data['ocrFixtures']),24)
        self.assertEqual(sum(r['count'] for r in data['rounds']),1784)
        keys=[(c['run'],c['variant'],c['id']) for c in data['cases']]
        self.assertEqual(len(keys),len(set(keys)))
        self.assertTrue(all(c['scene'] and c['effectiveScene'] and c['template'] for c in data['cases']))
        self.assertTrue(all('未测自动命中' in c['assignment'] for c in data['cases']))
    def test_report_loaders_preserve_module_and_path(self):
        sentinel=types.ModuleType('run');previous=sys.modules.get('run');before=list(sys.path)
        sys.modules['run']=sentinel
        try:
            self.assertEqual(len(report.collect()['cases']),1784)
            self.assertIs(sys.modules['run'],sentinel)
            self.assertEqual(sys.path,before)
        finally:
            sys.modules.pop('run',None)
            if previous is not None:sys.modules['run']=previous
    def test_hostile_text_remains_data(self):
        attack={'text':'</script><script>alert("wrong")</script>&OCR'}
        encoded=report.encode_data(attack)
        self.assertNotIn('</script>',encoded)
        self.assertNotIn('<',encoded)
        self.assertEqual(json.loads(encoded),attack)
    def test_strict_delivery_origin(self):
        data=report.collect()
        for item in data['cases']:
            if item['delivery']:
                accepted=item['strict']=='未启用' or (item['strict']=='accepted' and item['envelope']=='accepted')
                self.assertEqual(item['delivery']['text'],item['output'] if accepted else item['cleanup'])
if __name__=='__main__':unittest.main()
