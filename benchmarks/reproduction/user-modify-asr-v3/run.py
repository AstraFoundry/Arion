"""Actual ASR -> authored user edit -> production vault -> future audio ASR."""
import argparse
import importlib.metadata
import json
from pathlib import Path
import random
import resource
import shutil
import subprocess
import sys
import time
import wave

from common import ROOT, read, sha, source_hashes, v2_runner, verify_freeze, write
import retrieval


def main():
    parser=argparse.ArgumentParser()
    for name in ["fixtures","split","model","driver","out"]:
        parser.add_argument("--"+name,required=True)
    parser.add_argument("--selection")
    a=parser.parse_args()
    if a.split not in ["dev","test"]:
        raise ValueError("dev or test required")
    if (a.split=="test") != bool(a.selection):
        raise ValueError("Final test requires the frozen development selection")
    fixtures=Path(a.fixtures).resolve()
    verify_freeze(fixtures)
    config=read(fixtures/"config.json")
    selection=read(a.selection) if a.selection else None
    if selection and (selection["sourceHashes"]!=source_hashes() or selection["configSha256"]!=sha(fixtures/"config.json") or selection["initialFreezeSha256"]!=sha(fixtures/"freeze.json")):
        raise ValueError("Selected policy or initial freeze changed")
    arms=config["arms"] if not selection else list(dict.fromkeys(["none","v2_hint",selection["selectedArm"]]))
    split=fixtures/a.split
    # The runner never opens evaluation.json. Prior user corrections are an
    # explicit allowed teaching input; recurrence labels are evaluator-only.
    teaching=read(split/"teaching.json")["events"]
    future=read(split/"recognition.json")["samples"]
    controls=read(split/"edit-controls.json")["events"]
    for s in teaching+future:
        if sha(split/s["audio"]["path"])!=s["audio"]["sha256"]:
            raise ValueError("Audio identity changed")
    out=Path(a.out).resolve()
    out.mkdir(parents=True,exist_ok=False)
    driver=Path(a.driver).resolve()
    run=dict(status="running",kind="closed-loop-memory-v3",split=a.split,arms=arms,config=config,
             initialFreezeSha256=sha(fixtures/"freeze.json"),selectionSha256=sha(a.selection) if a.selection else None,
             sourceHashes=source_hashes(),driverSha256=sha(driver),
             inputHashes={str(p.relative_to(fixtures)):sha(p) for p in [fixtures/"config.json",fixtures/"selection-criteria.json",split/"teaching.json",split/"recognition.json",split/"edit-controls.json"]},
             revision=subprocess.check_output(["git","rev-parse","HEAD"],text=True).strip(),dirty=subprocess.check_output(["git","status","--porcelain"],text=True),
             hardware=dict(os=subprocess.check_output(["sw_vers"],text=True),cpu=subprocess.check_output(["sysctl","-n","machdep.cpu.brand_string"],text=True).strip(),ramBytes=int(subprocess.check_output(["sysctl","-n","hw.memsize"],text=True)),powerMode=subprocess.check_output(["pmset","-g"],text=True)),
             runtime={d.metadata["Name"]:d.version for d in importlib.metadata.distributions()},
             limitations=["Authored full-field user corrections, not observed user behavior or native edit tracking.","Teaching recognizes with memory disabled to collect independent prior-session error aliases.","Different teaching/recurrence voices; entities, audio, speakers and sessions disjoint across development/test.","Memory freezes before all recurrence; no future outputs, references or counters feed learning.","Community MLX 8-bit source BF16 parity unmeasured; offline complete audio, no native insertion or microphone timing."])
    def save():write(out/"run.json",run)
    save()
    try:
        import mlx.core as mx
        import mlx_audio
        from mlx_audio.stt import load
        import numpy as np
        import pypinyin
        import editdistance
        run["modelHashes"],run["modelRevision"]=v2_runner.verify_model(Path(a.model).resolve(),fixtures)
        if run["modelRevision"]!=config["revision"]:
            raise ValueError("Model revision mismatch")
        run["runtimeArtifactHashes"]={str(f):sha(f) for module in [mx,mlx_audio,pypinyin,editdistance] for f in Path(module.__file__).parent.rglob("*") if f.is_file() and f.suffix in [".py",".so",".dylib",".metallib"]}
        run["goVersion"]=subprocess.check_output(["go","version"],text=True).strip()
        run["driverBuild"]=subprocess.check_output(["go","version","-m",str(driver)],text=True)
        run["pythonSha256"]=sha(sys.executable)
        mx.random.seed(config["seed"])
        started=time.perf_counter()
        model=load(str(Path(a.model).resolve()))
        run["loadMs"]=(time.perf_counter()-started)*1000
        def audio_for(p):
            with wave.open(str(p),"rb") as f:
                if (f.getframerate(),f.getnchannels(),f.getsampwidth())!=(16000,1,2):
                    raise ValueError("Expected 16k mono PCM16")
                return np.frombuffer(f.readframes(f.getnframes()),dtype="<i2").astype(np.float32)/32768
        def infer(audio,context):
            mx.random.seed(config["seed"])
            mx.synchronize()
            started=time.perf_counter()
            r=model.generate(audio,system_prompt=context,max_tokens=config["maxTokens"],temperature=0,verbose=False)
            mx.synchronize()
            return dict(text=r.text,elapsedMs=(time.perf_counter()-started)*1000,promptTokens=r.prompt_tokens,generationTokens=r.generation_tokens,tokenLimitReached=r.generation_tokens>=config["maxTokens"],peakRssBytes=resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,peakMlxBytes=mx.get_peak_memory())
        warmup=fixtures/"warmup.wav"
        if sha(warmup)!=read(fixtures/"warmup.json")["sha256"]:
            raise ValueError("Warmup changed")
        with (out/"warmups.jsonl").open("x") as f:
            for context in ["","Vocabulary: calendar."]:
                r=infer(audio_for(warmup),context)
                f.write(json.dumps(dict(context=context,**r),ensure_ascii=False)+"\n")
        key=out/"private-vault.key"
        vault=out/"vocabulary.enc"
        def store(operation,**kwargs):
            request=dict(operation=operation,storePath=str(vault),keyPath=str(key),**kwargs)
            result=subprocess.run([str(driver)],input=json.dumps(request,ensure_ascii=False),text=True,capture_output=True,check=True)
            return json.loads(result.stdout)
        initial=store("init")
        if initial["afterCount"]!=0:
            raise ValueError("Expected empty initial memory")
        write(out/"initial-memory.json",initial)
        with (out/"calls.jsonl").open("x") as calls,(out/"teaching.jsonl").open("x") as teach_out,(out/"results.jsonl").open("x") as rows:
            count=0
            def call(case,stage,audio,context):
                nonlocal count
                count+=1
                r=infer(audio,context)
                r.update(callId=count,caseId=case,stage=stage,context=context)
                calls.write(json.dumps(r,ensure_ascii=False)+"\n");calls.flush()
                return r
            for i,event in enumerate(teaching):
                r=call(event["id"],"teaching",audio_for(split/event["audio"]["path"]),"")
                prefix,suffix="Entry: "," | end"
                edit=dict(Inserted=r["text"],Baseline=prefix+r["text"]+suffix,Settled=prefix+event["userEditedText"]+suffix)
                persisted=store("settle",edit=edit,unixTime=1800000000+i*60)
                row=dict(id=event["id"],sessionId=event["sessionId"],callId=r["callId"],edit=edit,persistence=persisted)
                teach_out.write(json.dumps(row,ensure_ascii=False)+"\n");teach_out.flush()
                print("teach",event["id"],persisted["extraction"]["Verdict"],persisted["afterCount"],flush=True)
            for i,event in enumerate(controls):
                prefix=event.get("prefix","")
                edit=dict(Inserted=event["inserted"],Baseline=prefix+event["inserted"],Settled=event["settled"])
                persisted=store("settle",edit=edit,unixTime=1800010000+i*60)
                teach_out.write(json.dumps(dict(id="edit-"+event["id"],kind=event["kind"],edit=edit,persistence=persisted),ensure_ascii=False)+"\n");teach_out.flush()
            # A new process reopens the encrypted store. Only these reloaded
            # approved records can supply contexts for future audio.
            snapshot=store("snapshot")
            entries=snapshot["entries"] or []
            write(out/"frozen-memory.json",snapshot)
            run["frozenMemorySha256"]=sha(out/"frozen-memory.json")
            run["frozenCiphertextSha256"]=sha(vault) if vault.exists() else None
            rng=random.Random(config["seed"])
            for s in future:
                audio=audio_for(split/s["audio"]["path"])
                base=call(s["id"],"first_pass",audio,"")
                rows.write(json.dumps(dict(caseId=s["id"],arm="none",text=base["text"],firstPassText=base["text"],selected=[],context="",route="first_pass",callIds=[base["callId"]],retrievalMs=0,secondPassMs=0,pipelineMs=base["elapsedMs"],tokenLimitReached=base["tokenLimitReached"]),ensure_ascii=False)+"\n");rows.flush()
                others=[arm for arm in arms if arm!="none"]
                rng.shuffle(others)
                for arm in others:
                    started=time.perf_counter()
                    context,selected=retrieval.context_for(arm,base["text"],entries,config)
                    retrieval_ms=(time.perf_counter()-started)*1000
                    result=call(s["id"],arm,audio,context) if context else base
                    second=result["elapsedMs"] if context else 0
                    ids=[base["callId"],result["callId"]] if context else [base["callId"]]
                    row=dict(caseId=s["id"],arm=arm,text=result["text"],firstPassText=base["text"],selected=selected,context=context,route="audio_redecode" if context else "no_retrieval",callIds=ids,retrievalMs=retrieval_ms,secondPassMs=second,pipelineMs=base["elapsedMs"]+retrieval_ms+second,tokenLimitReached=result["tokenLimitReached"])
                    rows.write(json.dumps(row,ensure_ascii=False)+"\n");rows.flush()
                print("future",s["id"],flush=True)
        if (sha(vault) if vault.exists() else None)!=run["frozenCiphertextSha256"]:
            raise ValueError("Memory changed during recurrence")
        if entries:
            # Lifecycle checks use an explicit clone, preserving the actual
            # frozen research memory and paired inference treatments.
            clone=out/"lifecycle.enc"
            shutil.copy2(vault,clone)
            original=vault
            vault=clone
            removed=store("remove",entryId=entries[0]["id"])
            if removed["afterCount"]!=len(entries)-1:
                raise ValueError("Undo/reopen count mismatch")
            reverse=entries[-1]
            reverted=store("settle",edit=dict(Inserted="请确认"+reverse["to"]+"的记录已经保存。",Baseline="请确认"+reverse["to"]+"的记录已经保存。",Settled="请确认"+reverse["from"]+"的记录已经保存。"),unixTime=1800020000)
            write(out/"lifecycle.json",dict(undo=removed,reversal=reverted,scope="Cloned store after paired recurrence; synthetic lifecycle edits."))
            vault=original
        run.update(status="complete",teachingCases=len(teaching),editControlCases=len(controls),recurrenceCases=len(future),observations=len(future)*len(arms),actualModelCalls=count)
        run["outputHashes"]={name:sha(out/name) for name in ["initial-memory.json","teaching.jsonl","frozen-memory.json","results.jsonl","calls.jsonl","warmups.jsonl","lifecycle.json"] if (out/name).exists()}
        save()
    except BaseException as e:
        run.update(status="failed",error=type(e).__name__+": "+str(e))
        save()
        raise


if __name__=="__main__":
    main()
