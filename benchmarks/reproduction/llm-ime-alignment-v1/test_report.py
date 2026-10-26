"""Model-free adversarial checks of the retained evaluation boundary."""
import shutil, tempfile, unittest
from pathlib import Path
from study import ROOT, read, write, sha, lines, report, schedule_report

RUN=ROOT/'benchmarks/runs/llm-ime-alignment-v1'
RESULT=ROOT/'benchmarks/results/llm-ime-alignment-v1'
class ReportTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory();self.addCleanup(self.temp.cleanup)
        self.work=Path(self.temp.name)/'run';shutil.copytree(RUN/'gated',self.work)
    def observations(self):return lines(self.work/'inference/observations.jsonl')
    def save_observations(self,rows): (self.work/'inference/observations.jsonl').write_text(''.join(__import__('json').dumps(r,ensure_ascii=False)+'\n' for r in rows))
    def test_exact_replay(self):self.assertEqual(report(self.work),read(RESULT/'gated-summary.json'))
    def test_invented_id_and_nonfinite_score(self):
        for mutate in [lambda r:r['result']['candidates'][0].update(id='invented'),lambda r:r['result']['candidates'][0].update(score=float('nan'))]:
            rows=self.observations();mutate(rows[2]);self.save_observations(rows)
            with self.assertRaises(ValueError):report(self.work)
            shutil.copyfile(RUN/'gated/inference/observations.jsonl',self.work/'inference/observations.jsonl')
    def test_incomplete_and_duplicate_observations(self):
        original=self.observations()
        for rows in [original[:-1],original[:2]+[original[2]]+original[3:-1]]:
            self.save_observations(rows)
            with self.assertRaises(ValueError):report(self.work)
    def test_failed_status(self):
        status=read(self.work/'inference/status.json');status['status']='failed';write(self.work/'inference/status.json',status)
        with self.assertRaises(ValueError):report(self.work)
    def test_label_leak_and_changed_producer(self):
        import json
        for key,value in [('acceptable',['背景']),('input','fabricated')]:
            reqs=lines(self.work/'requests.jsonl');reqs[2][key]=value
            (self.work/'requests.jsonl').write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in reqs))
            status=read(self.work/'inference/status.json');status['requestsSha256']=sha(self.work/'requests.jsonl');write(self.work/'inference/status.json',status)
            with self.assertRaises(ValueError):report(self.work)
            shutil.copyfile(RUN/'gated/requests.jsonl',self.work/'requests.jsonl');shutil.copyfile(RUN/'gated/inference/status.json',self.work/'inference/status.json')
    def test_scheduled_stale_application(self):
        import json
        rows=lines(RUN/'schedule-observations-v2.jsonl')
        row=next(r for r in rows if r['kind']=='outcome' and r['data']['error'])
        row['data']['canApply']=True
        out=Path(self.temp.name)/'stale.jsonl';out.write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in rows))
        with self.assertRaises(ValueError):schedule_report(RUN/'schedule.json',out)

if __name__=='__main__':unittest.main()
