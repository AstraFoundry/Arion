"""Select once on the disclosed development cases and guard controls."""
import argparse
from pathlib import Path
from run import source_hashes
from common import read,sha
import json

p=argparse.ArgumentParser()
for name in ["fixtures","dev-run","out"]:p.add_argument("--"+name,required=True)
a=p.parse_args()
fixtures=Path(a.fixtures);folder=Path(a.dev_run)
s=read(folder/"summary.json");r=read(folder/"run.json");protocol=read(fixtures/"protocol.json")
if s["status"]!="complete" or s["split"]!="dev" or source_hashes()!=r["sourceHashes"]:
    raise ValueError("Complete unchanged development run required")
eligible=[arm for arm in protocol["candidates"] if all(s["arms"][arm][k]==0 for k in ["controlFailures","correctInputRegressions","questionViolations","protectedViolations"])]
selected=min(eligible,key=lambda arm:(-s["arms"][arm]["exactMatches"],protocol["candidates"].index(arm))) if eligible else "product"
selection=dict(status="frozen_before_context_test",selectedArm=selected,eligibleArms=eligible,sourceHashes=r["sourceHashes"],freezeSha256=sha(fixtures/"freeze.json"),devSummarySha256=sha(folder/"summary.json"),devMetrics=s["arms"])
with Path(a.out).open("x") as f:f.write(json.dumps(selection,ensure_ascii=False,indent=2)+"\n")
print(selected)
