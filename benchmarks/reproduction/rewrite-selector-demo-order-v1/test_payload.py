"""Check complete original requests, candidate production and exact pair-only permutations."""
import copy,json,pathlib,unittest
from unittest.mock import patch
import pipeline
ROOT=pipeline.HERE.parents[2]
class PayloadTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.refs=[r for r in map(json.loads,(ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines()) if r['arm'] in ['baseline','balanced']]
        cls.preps={p['sample']['id']:p for p in map(json.loads,(ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/preparations.jsonl').read_text().splitlines())}
    def pairs(self):
        for r in self.refs:
            arm='negative-last-balanced' if r['arm']=='balanced' else 'negative-last';p=self.preps[r['id']]
            request,prepared=pipeline.selection_payload(arm,p['sample'],p['cleanupObservation']['cleanup'])
            self.assertEqual(prepared,p['prepared']);yield r,request,arm
    def test_complete_202_original_controls(self):
        self.assertEqual(len(self.refs),202)
        for r in self.refs:
            p=self.preps[r['id']];request,prepared=pipeline.selection_payload(r['arm'],p['sample'],p['cleanupObservation']['cleanup'])
            self.assertEqual(request,r['request']);self.assertEqual(prepared,p['prepared'])
    def test_exact_inverse_restores_complete_request(self):
        for r,request,arm in self.pairs():self.assertEqual(pipeline.restore_request(request,arm),r['request'])
    def test_all_ten_pairs_preserved_and_negative_examples_last(self):
        for r,request,_ in self.pairs():
            blocks=lambda ms:[ms[i:i+2] for i in range(1,21,2)]
            self.assertCountEqual(blocks(request['messages']),blocks(r['request']['messages']))
            self.assertEqual([m['content'] for m in request['messages'][-6:-1:2]],['{"e0":"keep"}']*3)
    def test_rules_final_source_and_settings_unchanged(self):
        for r,request,_ in self.pairs():
            self.assertEqual(request['messages'][0],r['request']['messages'][0]);self.assertEqual(request['messages'][-1],r['request']['messages'][-1])
            self.assertEqual({k:v for k,v in request.items() if k!='messages'},{k:v for k,v in r['request'].items() if k!='messages'})
    def test_projection_preserves_original_baseline(self):
        for r,request,arm in self.pairs():self.assertEqual(pipeline.baseline_projection(request,arm),pipeline.prior.baseline_projection(r['request'],r['arm']))
    def test_reordering_does_not_mutate_input(self):
        r=copy.deepcopy(self.refs[0]['request']);before=copy.deepcopy(r);pipeline.reorder(r);self.assertEqual(r,before)
    def test_unknown_arm_and_wrong_roles_fail(self):
        with self.assertRaises(ValueError):pipeline.control_arm('unknown')
        r=copy.deepcopy(self.refs[0]['request']);r['messages'][1]['role']='assistant'
        with self.assertRaises(ValueError):pipeline.reorder(r)
    def test_invalid_permutation_fails(self):
        original=pathlib.Path.read_text;path=pipeline.HERE/'layout.json';value=json.loads(original(path));value['pairPermutation']=[0]*10
        with patch.object(pathlib.Path,'read_text',lambda p,*a,**kw:json.dumps(value) if p==path else original(p,*a,**kw)):
            with self.assertRaises(ValueError):pipeline.reorder(self.refs[0]['request'])
if __name__=='__main__':unittest.main()
