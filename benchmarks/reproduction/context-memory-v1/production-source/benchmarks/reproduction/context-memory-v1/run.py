"""Actual text-LLM ablations using product prompt rendering, vault and validator."""
import argparse
import importlib.metadata
import json
from pathlib import Path
import random
import subprocess
import sys
import time

HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[2]
sys.path.insert(0,str(HERE.parent/"user-modify-asr-v3"))
from common import read,sha,write
import retrieval
from pypinyin import lazy_pinyin,Style


def source_hashes():
    files=[HERE/"run.py",ROOT/"apps/desktop/cmd/arion-context-probe/main.go",ROOT/"apps/desktop/cmd/arion-memory-loop/main.go",
           ROOT/"tools/evaluation/diagnostic/qwen_worker.py",HERE.parent/"user-modify-asr-v3/retrieval.py",HERE.parent/"user-modify-asr-v2/policy.py"]
    files+=list((ROOT/"packages/core/domain").glob("*.go"))
    files+=list((ROOT/"apps/desktop/infrastructure/llm/prompt").glob("*.json"))
    files+=[ROOT/"apps/desktop/infrastructure/llm/prompt/prompt.go",ROOT/"apps/desktop/infrastructure/vocabulary/vocabulary.go"]
    return {str(p.relative_to(ROOT)):sha(p) for p in files if not p.name.endswith("_test.go")}


def main():
    p=argparse.ArgumentParser()
    for name in ["fixtures","split","out","model","model-metadata","worker-python","memory-driver","context-driver"]:
        p.add_argument("--"+name,required=True)
    p.add_argument("--selection")
    a=p.parse_args()
    fixtures=Path(a.fixtures).resolve();out=Path(a.out).resolve()
    if a.split not in ["dev","test"] or ((a.split=="test")!=bool(a.selection)):
        raise ValueError("Final test requires a development-only selection")
    frozen=read(fixtures/"freeze.json")
    for name,m in frozen["files"].items():
        if sha(fixtures/name)!=m["sha256"]:
            raise ValueError("Frozen context fixture changed")
    protocol=read(fixtures/"protocol.json")
    selected=read(a.selection) if a.selection else None
    if selected and (selected["sourceHashes"]!=source_hashes() or selected["freezeSha256"]!=sha(fixtures/"freeze.json")):
        raise ValueError("Frozen prompt/inference policy changed")
    arms=protocol["arms"] if not selected else list(dict.fromkeys(["product","sections",selected["selectedArm"]]))
    samples=read(fixtures/a.split/"inputs.json")["samples"]
    feedback=read(fixtures/a.split/"feedback.json")["events"]
    out.mkdir(parents=True,exist_ok=False)
    model=Path(a.model).resolve()
    pinned=read(a.model_metadata)
    assets=[x for x in pinned["artifacts"] if Path(x["path"]).parent.name=="qwen3-1.7b"]
    if len(assets)!=6:
        raise ValueError("Expected six pinned text-model assets")
    for x in assets:
        if sha(x["path"])!=x["sha256"]:
            raise ValueError("Pinned text model changed")
    effective=[]
    for f in model.iterdir():
        if f.is_file():effective.append(dict(path=str(f),sha256=sha(f),sizeBytes=f.stat().st_size))
    if sha(model/"model.onnx")!=next(x["sha256"] for x in assets if Path(x["path"]).name=="model.onnx"):
        raise ValueError("Effective weights differ")
    key=out/"private-vault.key";vault=out/"vocabulary.enc"
    def native(binary,request):
        r=subprocess.run([str(Path(binary).resolve())],input=json.dumps(request,ensure_ascii=False),text=True,capture_output=True,check=True)
        return json.loads(r.stdout)
    def store(op,**kw):return native(a.memory_driver,dict(operation=op,storePath=str(vault),keyPath=str(key),**kw))
    run=dict(status="running",kind="context-memory-v1",evidence=protocol["evidence"],split=a.split,arms=arms,
             freezeSha256=sha(fixtures/"freeze.json"),sourceHashes=source_hashes(),selectionSha256=sha(a.selection) if a.selection else None,
             assets=assets,effectiveModelAssets=effective,modelMetadataSha256=sha(a.model_metadata),
             driverHashes={str(Path(x).resolve()):sha(x) for x in [a.memory_driver,a.context_driver,a.worker_python]},
             revision=subprocess.check_output(["git","rev-parse","HEAD"],text=True).strip(),dirty=subprocess.check_output(["git","status","--porcelain"],text=True),
             hardware=dict(os=subprocess.check_output(["sw_vers"],text=True),cpu=subprocess.check_output(["sysctl","-n","machdep.cpu.brand_string"],text=True).strip(),ramBytes=int(subprocess.check_output(["sysctl","-n","hw.memsize"],text=True)),powerMode=subprocess.check_output(["pmset","-g"],text=True)),
             localRuntime={d.metadata["Name"]:d.version for d in importlib.metadata.distributions()})
    save=lambda:write(out/"run.json",run)
    save();worker=None
    try:
        initial=store("init")
        if initial["afterCount"]!=0:raise ValueError("Expected empty memory")
        write(out/"initial-memory.json",initial)
        with (out/"teaching.jsonl").open("x") as f:
            for i,e in enumerate(feedback):
                result=store("settle",edit=dict(Inserted=e["inserted"],Baseline=e["inserted"],Settled=e["settled"]),unixTime=1800000000+i*60)
                f.write(json.dumps(dict(id=e["id"],feedback=e,persistence=result),ensure_ascii=False)+"\n")
        snapshot=store("snapshot")
        write(out/"frozen-memory.json",snapshot)
        entries=snapshot["entries"] or []
        run["frozenCiphertextSha256"]=snapshot.get("ciphertextSha256")
        config=dict(logicalDay=100,expireDays=90,retrievalLimit=2,fuzzyThreshold=0.25)
        worker=subprocess.Popen([a.worker_python,str(ROOT/"tools/evaluation/diagnostic/qwen_worker.py")],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=(out/"worker-stderr.log").open("x"),text=True,bufsize=1)
        seq=0
        def ask(request):
            nonlocal seq
            seq+=1;request["id"]=seq
            worker.stdin.write(json.dumps(request,ensure_ascii=False)+"\n");worker.stdin.flush()
            line=worker.stdout.readline()
            if not line:raise RuntimeError("Rewrite worker exited")
            result=json.loads(line)
            if not result.get("ok") or result.get("id")!=seq:raise RuntimeError(str(result))
            return result
        run["workerRuntime"]=ask(dict(op="load",modelPath=str(model),artifacts=effective))
        warm=ask(dict(op="generate",prompt="<|im_start|>system\nRepeat the input.<|im_end|>\n<|im_start|>user\nReady.<|im_end|>\n<|im_start|>assistant\n<think>\n\n</think>\n\n",maxInputTokens=4096,maxNewTokens=32))
        write(out/"warmup.json",warm)
        jobs=[(s,arm) for s in samples for arm in arms]
        random.Random(protocol["seed"]).shuffle(jobs)
        semantic=read(fixtures/"semantic-template.json")
        with (out/"results.jsonl").open("x") as f:
            for s,arm in jobs:
                _,hits=retrieval.context_for("v2_hint",s["text"],entries,config)
                records=[next(e for e in entries if e["id"]==h["id"]) for h in hits]
                terms=[e["from"]+" → "+e["to"] for e in records]
                cases=[]
                if arm=="phonetic_context":
                    for e,h in zip(records,hits):
                        tone=lambda text:" ".join(lazy_pinyin(text,style=Style.TONE3,neutral_tone_with_five=True))
                        case=dict(observed=e["from"],observedPinyin=tone(e["from"]),corrected=e["to"],correctedPinyin=tone(e["to"]),queryDistance=h["score"])
                        cases.append(case)
                    terms=[json.dumps(c,ensure_ascii=False) for c in cases]
                request=dict(operation="render",mode=arm,text=s["text"],context=s["context"],entries=entries,terms=terms)
                if arm in ["semantic_context","phonetic_context"]:request["template"]=semantic
                rendered=native(a.context_driver,request)
                response=ask(dict(op="generate",prompt=rendered["prompt"],maxInputTokens=protocol["maxInputTokens"],maxNewTokens=protocol["maxNewTokens"]))
                validation=native(a.context_driver,dict(request,operation="validate",candidate=response["text"],truncated=response["truncated"]))
                delivered=response["text"] if validation["verdict"]=="accepted" else rendered["source"]
                row=dict(caseId=s["id"],arm=arm,input=s,selected=hits,correctionExamples=cases,rendered=rendered,response=response,validation=validation,delivered=delivered)
                f.write(json.dumps(row,ensure_ascii=False)+"\n");f.flush()
                print(s["id"],arm,validation["verdict"],flush=True)
        ask(dict(op="close"));worker.stdin.close();worker.wait(timeout=30)
        if worker.returncode!=0:raise RuntimeError("Rewrite worker failed on close")
        if sha(vault)!=run["frozenCiphertextSha256"]:raise ValueError("Memory changed during rewrite")
        run.update(status="complete",cases=len(samples),observations=len(jobs),actualModelCalls=len(jobs),warmups=1)
        run["outputHashes"]={n:sha(out/n) for n in ["initial-memory.json","teaching.jsonl","frozen-memory.json","results.jsonl","warmup.json","vocabulary.enc","worker-stderr.log"]}
        save()
    except BaseException as e:
        if worker is not None and worker.poll() is None:worker.terminate();worker.wait(timeout=30)
        run.update(status="failed",error=type(e).__name__+": "+str(e));save();raise


if __name__=="__main__":main()
