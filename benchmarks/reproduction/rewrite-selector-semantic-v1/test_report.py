"""Test actual evidence corruption and future prompt ownership without fabricated inference."""
import copy,unittest,pathlib,sys,types
import report
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parent.parent))
from gguf_template_identity import identity
class SemanticEvidenceTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.rows,cls.labels=report.checked();cls.m=report.read(report.RUN/'manifest.json');cls.j=report.read(report.RUN/'review.json');cls.o=list(map(report.json.loads,(report.RUN/'case-outcomes.jsonl').read_text().splitlines()));cls.d=report.read(report.RUN/'selection.json')
 def corrupt(self,edit):
  rows,m,j,o,d=map(copy.deepcopy,[self.rows,self.m,self.j,self.o,self.d]);edit(rows,m,j,o,d);labels={(x['variant'],x['id']):x for x in j['reviews']}
  for r in rows:
   if (r['variant'],r['id']) in labels:labels[(r['variant'],r['id'])]['rowSha256']=report.rowhash(r)
  with self.assertRaises((ValueError,KeyError)):report.validate(rows,m,j,o,d)
 def test_actual_scope(self):self.assertEqual((len(self.rows),self.m['formatterGenerations'],self.m['selectorGenerations']),(609,489,120))
 def test_missing_body(self):self.corrupt(lambda r,m,j,o,d:r.pop())
 def test_missing_source(self):self.corrupt(lambda r,m,j,o,d:o.pop())
 def test_changed_actual_prompt(self):self.corrupt(lambda r,m,j,o,d:next(x for x in r if x['stage']=='selection')['request']['messages'][0].update(content='wrong policy'))
 def test_filtered_body(self):self.corrupt(lambda r,m,j,o,d:r[0].update(renderedBody='filtered'))
 def test_false_adoption(self):self.corrupt(lambda r,m,j,o,d:d.update(qualified=True))
 def test_user_same_letter_policy(self):
  j=next(x for x in self.j['reviews'] if x['id']=='tone-control-18' and x['stage']=='selection');self.assertTrue(j['choiceConforms'])
 def test_future_explicit_prompt_identity(self):
  f=report.read(report.HERE/'freeze.json');p=types.SimpleNamespace(template_identity=lambda *args:dict(version='semantic-v1',path='benchmarks/reproduction/rewrite-selector-semantic-v1/prompts.json'));request=next(r['request'] for r in self.rows if r['arm']=='candidate' and r['stage']=='selection');v=identity(report.ROOT,f,p,'candidate','selection','work',request);self.assertEqual(v['templateSha256'],report.sha(report.HERE/'prompts.json'))
  with self.assertRaises(ValueError):identity(report.ROOT,dict(files={}),p,'candidate','selection','work',request)
if __name__=='__main__':unittest.main()
