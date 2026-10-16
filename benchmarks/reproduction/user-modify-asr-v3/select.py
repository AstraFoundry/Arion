"""Select at most two predeclared development candidates; never inspect test."""
import argparse
from pathlib import Path
from common import read, sha, source_hashes, write

p=argparse.ArgumentParser()
for name in ["fixtures","dev-run","out"]:
    p.add_argument("--"+name,required=True)
a=p.parse_args()
fixtures=Path(a.fixtures)
dev=Path(a.dev_run)
s=read(dev/"summary.json")
r=read(dev/"run.json")
criteria=read(fixtures/"selection-criteria.json")
if s["status"]!="complete" or s["split"]!="dev" or r["sourceHashes"]!=source_hashes():
    raise ValueError("Incomplete development or changed inference policy")
baseline=criteria["baseline"]
rank=lambda arm:(-s["arms"][arm]["primaryEntityHits"],s["arms"][arm]["lexicalErrors"],s["arms"][arm]["pipelineMs"]["p95"],([baseline]+criteria["candidates"]).index(arm))
eligible=[arm for arm in criteria["candidates"] if all(s["arms"][arm][metric]==0 for metric in ["newWrongEntityCases","baselineCorrectRegressed","nonspeechEmissions","controlWorsened"])]
winner=min([baseline]+eligible,key=rank)
selection=dict(status="frozen_before_test_teaching_and_recognition",selectedArm=winner,eligibleArms=eligible,
               sourceHashes=r["sourceHashes"],configSha256=sha(fixtures/"config.json"),initialFreezeSha256=sha(fixtures/"freeze.json"),
               selectionCriteriaSha256=sha(fixtures/"selection-criteria.json"),devSummarySha256=sha(dev/"summary.json"),
               reason="Retained previous policy" if winner==baseline else "Candidate passed development guards and outranked the previous policy",
               devMetrics=s["arms"])
out=Path(a.out)
with out.open("x") as f:
    import json
    f.write(json.dumps(selection,ensure_ascii=False,indent=2)+"\n")
print(winner)
