"""Freeze entity-disjoint authored teaching and recurrence sessions before tuning."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import wave

import numpy as np


def sha(p):
    return hashlib.sha256(Path(p).read_bytes()).hexdigest()


def write(p, value):
    p.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n")


DEV = [
    ("PostHog", "破斯特霍格", "mixed"),
    ("Temporal", "坦普若", "mixed"),
    ("Supabase", "苏帕贝斯", "mixed"),
    ("Drizzle", "追兹尔", "mixed"),
    ("Convex", "康维克斯", "mixed"),
    ("Turso", "特索", "mixed"),
    ("Svelte", "斯维尔特", "mixed"),
    ("Vite", "维特", "mixed"),
    ("顾珩川", "顾珩川", "zh"),
    ("温栩然", "温栩然", "zh"),
    ("棠溪", "棠溪", "zh"),
    ("霁禾", "霁禾", "zh"),
    ("Aisling", "Aisling", "en"),
    ("Eoin", "Eoin", "en"),
    ("Cillian", "Cillian", "en"),
    ("Maeve", "Maeve", "en"),
]
TEST = [
    ("ClickHouse", "克里克豪斯", "mixed"),
    ("Prometheus", "普罗米修斯", "mixed"),
    ("Grafana", "格拉法纳", "mixed"),
    ("Milvus", "米尔沃斯", "mixed"),
    ("Weaviate", "维维艾特", "mixed"),
    ("DuckDB", "达克迪比", "mixed"),
    ("Raycast", "瑞卡斯特", "mixed"),
    ("Obsidian", "奥布西迪安", "mixed"),
    ("Zed", "泽德", "mixed"),
    ("Cursor", "科瑟", "mixed"),
    ("Ollama", "欧拉马", "mixed"),
    ("Tauri", "陶里", "mixed"),
    ("祁砚舟", "祁砚舟", "zh"),
    ("邵栩宁", "邵栩宁", "zh"),
    ("苏芷澄", "苏芷澄", "zh"),
    ("蓝泊川", "蓝泊川", "zh"),
    ("霁川", "霁川", "zh"),
    ("栖禾", "栖禾", "zh"),
    ("Siobhan", "Siobhan", "en"),
    ("Niamh", "Niamh", "en"),
    ("Aoife", "Aoife", "en"),
    ("Saoirse", "Saoirse", "en"),
    ("Joaquin", "Joaquin", "en"),
    ("Xochitl", "Xochitl", "en"),
]

TEMPLATES = {
    "mixed": {
        "teach": ["请把{t}加入今天的技术清单。", "这次分享的主题是{t}，材料已经准备好了。"],
        "recurrence": ["我们需要重新检查{t}的配置文件。", "请确认{t}有没有完成更新。", "关于{t}的问题留到下午讨论。"],
    },
    "zh": {
        "teach": ["请把{t}加入今天的参会名单。", "这次讨论由{t}主持，材料已经准备好了。"],
        "recurrence": ["明天请{t}带上会议记录。", "我已经给{t}发过确认消息了。", "关于这份材料，先听听{t}的意见。"],
    },
    "en": {
        "teach": ["Please add {t} to the meeting invitation today.", "I spoke with {t} about the schedule this morning."],
        "recurrence": ["Could you ask {t} to review the document?", "We are still waiting for a reply from {t}.", "The next presentation will be prepared by {t}."],
    },
}

CONTROLS = {
    "dev": [
        ("zh", "门口的师傅正在给窗户装玻璃。"),
        ("zh", "小朋友喜欢沿着溪水寻找石头。"),
        ("zh", "我们把温水倒进透明的杯子。"),
        ("zh", "明天周一先整理会议材料。"),
        ("zh", "这张表格需要重新核对一遍。"),
        ("zh", "请在打印之前确认纸张大小。"),
        ("en", "The farmer walked slowly towards the gate."),
        ("en", "She placed the glass beside the window."),
        ("en", "Please leave enough room for the next paragraph."),
        ("en", "I think the meeting starts after lunch."),
        ("en", "The water is too cold for swimming."),
        ("en", "We should finish the report before dinner."),
    ],
    "test": [
        ("zh", "小船停在水边，等大家吃完午饭再出发。"),
        ("zh", "他喜欢骑车去河边看日落。"),
        ("zh", "请把米放进锅里再加水。"),
        ("zh", "明天周一要记得早点出门。"),
        ("zh", "这件外套的拉链好像坏了。"),
        ("zh", "桌子上的咖啡已经凉了。"),
        ("zh", "这位老师讲课的时候语速很慢。"),
        ("zh", "栖息在树上的鸟儿还没有醒来。"),
        ("en", "I found a cursor shaped like a small arrow."),
        ("en", "The duck swam quietly across the pond."),
        ("en", "She drew a simple chart on the board."),
        ("en", "The coast is covered in thick fog this morning."),
        ("en", "Please move the chair away from the door."),
        ("en", "We need another copy of this document."),
        ("en", "The children are playing outside the library."),
        ("en", "He left a short note on the kitchen table."),
    ],
}


def edit_controls(split):
    marker = "甲" if split == "dev" else "乙"
    return [
        dict(id="unchanged", kind="negative", inserted="材料已经准备好了。", settled="材料已经准备好了。"),
        dict(id="append", kind="negative", inserted="材料已经准备好了。", settled="材料已经准备好了。明天再发。"),
        dict(id="punctuation", kind="negative", inserted="材料已经准备好了。", settled="材料已经准备好了！"),
        dict(id="number", kind="negative", inserted="请在九点参加会议。", settled="请在十点参加会议。"),
        dict(id="grammar", kind="negative", inserted="他的材料已经准备好了。", settled="她的材料已经准备好了。"),
        dict(id="clear", kind="negative", inserted="材料已经准备好了。", settled=""),
        dict(id="delete", kind="negative", inserted="材料已经全部准备好了。", settled="材料已经准备好了。"),
        dict(id="outside", kind="negative", inserted="材料已经准备好了。", prefix=marker+"组：", settled=marker+"队：材料已经准备好了。"),
        dict(id="rewrite", kind="negative", inserted="下午先讨论演示文稿的修改意见。", settled="明天请大家早点出发。"),
        dict(id="weekday-intent", kind="semantic_risk", inserted="明天周一先整理材料。", settled="明天周二先整理材料。"),
    ]


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--out", required=True)
    a = p.parse_args()
    root = Path(a.out)
    root.mkdir(parents=True, exist_ok=False)
    old = Path("benchmarks/corpora/user-modify-asr-v1")
    for name in ["model-metadata.json", "all-model-files.json", "warmup.wav", "warmup.aiff", "warmup.json"]:
        shutil.copy2(old/name, root/name)
    voices = {
        "dev": {"teach": {"zh": "Tingting", "en": "Samantha"}, "recurrence": {"zh": "Eddy (中文（中国大陆）)", "en": "Karen"}},
        "test": {"teach": {"zh": "Flo (中文（中国大陆）)", "en": "Moira"}, "recurrence": {"zh": "Reed (中文（中国大陆）)", "en": "Tessa"}},
    }
    terms = {"dev": DEV, "test": TEST}
    assert not set(t[0].casefold() for t in DEV) & set(t[0].casefold() for t in TEST)
    count = 0
    def audio(out, ident, text, voice):
        nonlocal count
        aiff, wav = out/(ident+".aiff"), out/(ident+".wav")
        subprocess.run(["say", "-v", voice, "-r", "165", "-o", str(aiff), text], check=True)
        subprocess.run(["ffmpeg", "-loglevel", "error", "-i", str(aiff), "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", str(wav)], check=True)
        with wave.open(str(wav), "rb") as f:
            duration = f.getnframes()/f.getframerate()*1000
        count += 1
        if count % 20 == 0: print("built", count, flush=True)
        return dict(path=wav.name, sha256=sha(wav), durationMs=duration, speakerId=voice)
    for split in ["dev", "test"]:
        out = root/split
        out.mkdir()
        teaching, recognition, evaluation, teach_labels = [], [], [], []
        for i, (term, spoken, language) in enumerate(terms[split]):
            for j, template in enumerate(TEMPLATES[language]["teach"]):
                ident = f"{split}-teach-{i:02d}-{j}"
                voice = voices[split]["teach"]["en" if language == "en" else "zh"]
                wav = audio(out, ident, template.format(t=spoken), voice)
                teaching.append(dict(id=ident, sessionId=f"{split}-prior-{j}", audio=wav, userEditedText=template.format(t=term)))
                teach_labels.append(dict(id=ident, term=term, language=language))
            for j, template in enumerate(TEMPLATES[language]["recurrence"]):
                ident = f"{split}-recur-{i:02d}-{j}"
                voice = voices[split]["recurrence"]["en" if language == "en" else "zh"]
                wav = audio(out, ident, template.format(t=spoken), voice)
                recognition.append(dict(id=ident, sessionId=f"{split}-future-{j}", audio=wav))
                evaluation.append(dict(id=ident, term=term, language=language, unit={"zh":"cer", "en":"wer", "mixed":"mer"}[language], kind="recurring", reference=template.format(t=term)))
        for i, (language, text) in enumerate(CONTROLS[split]):
            ident = f"{split}-control-{i:02d}"
            wav = audio(out, ident, text, voices[split]["recurrence"][language])
            recognition.append(dict(id=ident, sessionId=split+"-controls", audio=wav))
            evaluation.append(dict(id=ident, term="", language=language, unit="cer" if language=="zh" else "wer", kind="ordinary_control", reference=text))
        unknown = [("Fennel", "费内尔"), ("Deno", "迪诺")] if split=="dev" else [("Nushell", "努谢尔"), ("Valkey", "瓦尔基"), ("NixOS", "尼克斯欧艾斯")]
        for i,(term,spoken) in enumerate(unknown):
            for j,template in enumerate(["请检查{t}的设置有没有保存。", "明天再讨论{t}的使用方法。"]):
                ident=f"{split}-unlearned-{i:02d}-{j}"
                wav=audio(out,ident,template.format(t=spoken),voices[split]["recurrence"]["zh"])
                recognition.append(dict(id=ident,sessionId=split+"-unlearned",audio=wav))
                evaluation.append(dict(id=ident,term=term,language="mixed",unit="mer",kind="unlearned_control",reference=template.format(t=term)))
        if split == "test":
            source=Path("benchmarks/corpora/asr-v1")
            corpus=json.loads((source/"corpus.json").read_text())
            natural=[]
            for language in ["zh","en"]:
                chosen=sorted([s for s in corpus["samples"] if s["split"]=="test" and s["language"]==language],key=lambda s:s["id"])[10:20]
                for s in chosen:
                    src=source/s["audio"]["path"]
                    assert sha(src)==s["audio"]["sha256"]
                    ident="natural-"+s["id"]
                    shutil.copy2(src,out/(ident+".wav"))
                    wav=dict(path=ident+".wav",sha256=sha(src),durationMs=s["audio"]["durationMs"],speakerId=s["speakerId"])
                    recognition.append(dict(id=ident,sessionId="historical-"+s["sessionId"],audio=wav))
                    evaluation.append(dict(id=ident,term="",language=language,unit=s["unit"],kind="natural_control",reference=s["reference"]))
                    natural.append(s)
            write(out/"natural-provenance.json",dict(source="benchmarks/corpora/asr-v1",sourceCorpusSha256=sha(source/"corpus.json"),selection="sorted test IDs 10:20 per language; disjoint from v2 controls",samples=natural))
        rng=np.random.default_rng(3021 if split=="dev" else 3022)
        for ident,pcm in [("silence",np.zeros(32000,dtype="<i2")),("noise",rng.normal(0,160,32000).astype("<i2"))]:
            wav=out/(ident+".wav")
            with wave.open(str(wav),"wb") as f:
                f.setnchannels(1);f.setsampwidth(2);f.setframerate(16000);f.writeframes(pcm.tobytes())
            case=split+"-"+ident
            recognition.append(dict(id=case,sessionId=case,audio=dict(path=wav.name,sha256=sha(wav),durationMs=2000,speakerId=split+"-generated-nonspeech")))
            evaluation.append(dict(id=case,term="",language="zh",unit="cer",kind="nonspeech",reference=""))
        write(out/"teaching.json",dict(events=teaching))
        write(out/"recognition.json",dict(samples=recognition))
        write(out/"evaluation.json",dict(teaching=teach_labels,samples=evaluation))
        write(out/"edit-controls.json",dict(events=edit_controls(split)))
        write(out/"provenance.json",dict(kind="authored_closed_loop_memory",initialMemoryEntries=0,terms=len(terms[split]),teaching=len(teaching),recognition=len(recognition),voices=voices[split],source="Developer-authored text and system TTS; simulated user corrections of actual ASR output",limitations=["No real user edit behavior or native editor tracking.","Transliterated technical pronunciations are authored and may be artificial.","English name pronunciation depends on the system voice.","Natural reads are historical regression, not natural recurrence."],builderSha256=sha(__file__)))
    criteria=dict(candidates=["native_vocab","competing_vocab"],baseline="v2_hint",guards=["no_added_wrong_entities","no_baseline_zero_error_regressions","no_nonspeech_emissions","no_control_worsening"],ranking=["recurring_entity_hits_desc","all_lexical_errors_asc","pipeline_p95_asc","declared_order"],fallback="retain_v2_if_no_candidate_passes_or_beats_it",maxDevelopmentCandidates=2)
    write(root/"selection-criteria.json",criteria)
    write(root/"config.json",dict(id="user-modify-asr-v3",revision="a8379a2e2f9e313c9292cdf1af4055ab56d50d55",seed=3021,maxTokens=192,logicalDay=100,expireDays=90,retrievalLimit=2,fuzzyThreshold=0.25,competitionThreshold=0.34,competitionMargin=0.08,arms=["none","v2_hint","native_vocab","competing_vocab"],frozenMemoryDuringRecurrence=True))
    # No inference has occurred. These identities cover audio, labels, feedback,
    # negative edits, selection criteria and all effective thresholds.
    write(root/"freeze.json",dict(status="frozen_before_any_v3_inference",files={str(p.relative_to(root)):dict(sha256=sha(p),sizeBytes=p.stat().st_size) for p in sorted(root.rglob("*")) if p.is_file()}))
    print("frozen",count,"synthesized clips",flush=True)


if __name__=="__main__":
    main()
