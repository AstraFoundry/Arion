"""Build Arion's private evaluation corpus (DATA-01) from pinned sources.

Run with an isolated interpreter (python3 -I) that provides pyarrow, plus
ffmpeg and, for synthetic fixtures, macOS `say`. Inputs and outputs stay in
the ignored .workspace directory; only this script and the prompt list are
versioned. No audio is fetched here: download the pinned files first (see
docs/research/streaming-datasets.md).

Usage:
  python3 -I tools/evaluation/data/build_corpus.py --raw .workspace/corpus/raw --out .workspace/corpus/v1
"""

import argparse
import hashlib
import io
import json
import os
import random
import re
import subprocess
import sys
import wave

import pyarrow.parquet as pq

SAMPLE_RATE = 16000
AISHELL_DEV_SPEAKERS = {"S0764", "S0765", "S0766"}
AISHELL_TEST_SPEAKERS = {"S0767", "S0768", "S0769", "S0770"}
PER_SPLIT = {"zh": {"dev": 45, "test": 60}, "en": {"dev": 40, "test": 60}}
VOICES = {
    "train": ["Grandpa (中文（中国大陆）)"],
    "dev": ["Eddy (中文（中国大陆）)", "Flo (中文（中国大陆）)", "Reed (中文（中国大陆）)"],
    "test": ["Tingting", "Shelley (中文（中国大陆）)", "Rocko (中文（中国大陆）)", "Sandy (中文（中国大陆）)"],
}


def sha256(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def to_wav(source_bytes, destination, source_path=None):
    """Decode any input deterministically to mono 16 kHz s16 WAV with ffmpeg."""
    command = ["ffmpeg", "-nostdin", "-loglevel", "error", "-y", "-i", source_path or "pipe:0",
               "-ac", "1", "-ar", str(SAMPLE_RATE), "-sample_fmt", "s16", "-bitexact", "-map_metadata", "-1", destination]
    subprocess.run(command, input=source_bytes, check=True)
    with wave.open(destination, "rb") as w:
        return round(w.getnframes() * 1000 / w.getframerate(), 3)


def sample(sample_id, split, language, speaker, session, tags, wav_path, out_dir, reference):
    unit = {"zh": "cer", "en": "wer", "mixed": "mer"}[language]
    with wave.open(wav_path, "rb") as w:
        duration = round(w.getnframes() * 1000 / w.getframerate(), 3)
    return {
        "id": sample_id, "split": split, "speakerId": speaker, "sessionId": session, "language": language, "unit": unit,
        "tags": tags, "audio": {"path": os.path.relpath(wav_path, out_dir), "sha256": sha256(wav_path), "durationMs": duration},
        "reference": reference, "context": "", "approvedMemory": [],
    }


def aishell(raw, out_dir, audio_dir):
    transcript = {}
    with open(os.path.join(raw, "aishell_transcript_v0.8.txt"), encoding="utf-8") as f:
        for line in f:
            utt, _, text = line.strip().partition(" ")
            transcript.setdefault(text.replace(" ", ""), []).append(utt)
    table = pq.read_table(os.path.join(raw, "aishell-test-mirror-00000.parquet"))
    rows = table.to_pylist()
    chosen = {"dev": [], "test": []}
    unmatched = 0
    for row in rows:
        text = row["answer"].replace(" ", "")
        ids = transcript.get(text, [])
        speakers = {utt[6:11] for utt in ids}
        # Speaker identity is recovered only when the official transcript maps
        # the text to exactly one test utterance (methodology: recover metadata).
        if len(ids) != 1:
            unmatched += 1
            continue
        utt, speaker = ids[0], ids[0][6:11]
        split = "dev" if speaker in AISHELL_DEV_SPEAKERS else "test" if speaker in AISHELL_TEST_SPEAKERS else None
        if split:
            chosen[split].append((utt, speaker, row, text))
    out = []
    rng = random.Random(1729)
    for split, items in chosen.items():
        items.sort(key=lambda item: item[0])
        rng.shuffle(items)
        for utt, speaker, row, text in items[: PER_SPLIT["zh"][split]]:
            path = os.path.join(audio_dir, f"aishell-{utt}.wav")
            to_wav(row["context"]["bytes"], path)
            out.append(sample(f"aishell-{utt}", split, "zh", f"aishell1:{speaker}", f"aishell1:{speaker}", ["read", "aishell1"], path, out_dir, text))
    return out, unmatched


def librispeech(raw, out_dir, audio_dir):
    out = []
    rng = random.Random(1729)
    for split, name in (("dev", "validation.clean.parquet"), ("test", "test.clean.parquet")):
        rows = pq.read_table(os.path.join(raw, name), columns=["id", "speaker_id", "chapter_id", "text", "audio"]).to_pylist()
        rows.sort(key=lambda r: r["id"])
        rng.shuffle(rows)
        per_speaker = {}
        picked = []
        for row in rows:
            words = len(row["text"].split())
            if not 6 <= words <= 30 or per_speaker.get(row["speaker_id"], 0) >= 2:
                continue
            per_speaker[row["speaker_id"]] = per_speaker.get(row["speaker_id"], 0) + 1
            picked.append(row)
            if len(picked) == PER_SPLIT["en"][split]:
                break
        for row in picked:
            path = os.path.join(audio_dir, f"libri-{row['id']}.wav")
            to_wav(row["audio"]["bytes"], path)
            out.append(sample(f"libri-{row['id']}", split, "en", f"librispeech:{row['speaker_id']}",
                              f"librispeech:{row['speaker_id']}-{row['chapter_id']}", ["read", "librispeech"], path, out_dir, row["text"].lower()))
    return out


def synthetic(prompts, out_dir, audio_dir):
    out = []
    for split, voices in VOICES.items():
        sentences = prompts[split]
        for index, text in enumerate(sentences):
            voice = voices[index % len(voices)]
            slug = re.sub(r"[^a-z]", "", voice.split(" ")[0].lower())
            sample_id = f"tts-{split}-{index:02d}-{slug}"
            aiff = os.path.join(audio_dir, sample_id + ".aiff")
            path = os.path.join(audio_dir, sample_id + ".wav")
            # The speech service occasionally stalls; retry a bounded number of times.
            for attempt in range(3):
                try:
                    subprocess.run(["say", "-v", voice, "-o", aiff, text], check=True, timeout=60)
                    break
                except subprocess.TimeoutExpired:
                    if attempt == 2:
                        raise
            to_wav(None, path, aiff)
            os.remove(aiff)
            out.append(sample(sample_id, split, "mixed", f"tts:{slug}", f"tts:{slug}:{split}", ["synthetic-tts", "technical"], path, out_dir, text))
    return out


def nonspeech(out_dir, audio_dir):
    out = []
    rng = random.Random(2718)
    for split, count in (("dev", 3), ("test", 4)):
        for index in range(count):
            sample_id = f"nonspeech-{split}-{index}"
            path = os.path.join(audio_dir, sample_id + ".wav")
            # Durations differ per split so no waveform appears in two splits.
            seconds = 2 + index + (0.5 if split == "test" else 0)
            amplitude = [0, 60, 250, 900][index % 4]
            frames = bytearray()
            for _ in range(int(SAMPLE_RATE * seconds)):
                frames += int(rng.gauss(0, amplitude)).to_bytes(2, "little", signed=True) if amplitude else b"\x00\x00"
            with wave.open(path, "wb") as w:
                w.setnchannels(1)
                w.setsampwidth(2)
                w.setframerate(SAMPLE_RATE)
                w.writeframes(bytes(frames))
            out.append(sample(sample_id, split, "zh", f"nonspeech:{split}", f"nonspeech:{split}:{index}", ["nonspeech", "noise" if amplitude else "silence"], path, out_dir, ""))
    return out


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--raw", required=True)
    parser.add_argument("--out", required=True)
    args = parser.parse_args()
    if os.path.exists(args.out):
        sys.exit("refusing to overwrite an existing corpus directory")
    audio_dir = os.path.join(args.out, "audio")
    os.makedirs(audio_dir)
    here = os.path.dirname(os.path.abspath(__file__))
    with open(os.path.join(here, "mixed_sentences.json"), encoding="utf-8") as f:
        prompts = json.load(f)
    zh, unmatched = aishell(args.raw, args.out, audio_dir)
    samples = zh + librispeech(args.raw, args.out, audio_dir) + synthetic(prompts, args.out, audio_dir) + nonspeech(args.out, audio_dir)
    sources = {name: sha256(os.path.join(args.raw, name)) for name in sorted(os.listdir(args.raw)) if not name.endswith(".part")}
    manifest = {
        "version": 1,
        "license": "AISHELL-1 Apache-2.0 (OpenSLR SLR33; HF mirror AudioLLMs/aishell_1_zh_test@e73dc7f0, transcript AISHELL/AISHELL-1@bbe295d5); "
                   "LibriSpeech CC-BY-4.0 (openslr/librispeech_asr@71cacbfb); synthetic macOS TTS prompts authored for Arion (CC0); generated nonspeech",
        "consent": True,
        "provenance": {
            "builder": "tools/evaluation/data/build_corpus.py", "sources": sources,
            "ffmpeg": subprocess.run(["ffmpeg", "-version"], capture_output=True, text=True).stdout.splitlines()[0],
            "aishellSpeakerRecovery": {"method": "unique transcript text match against official aishell_transcript_v0.8", "unmatchedRows": unmatched},
            "aishellSplitNote": "Official AISHELL-1 test speakers are partitioned by speaker into Arion dev (S0764-S0766) and test (S0767-S0770)",
            "syntheticNote": "TTS fixtures are a synthetic stress set; they are not natural speech",
        },
        "samples": samples,
    }
    with open(os.path.join(args.out, "corpus.json"), "w", encoding="utf-8") as f:
        json.dump(manifest, f, ensure_ascii=False, indent=1)
    counts = {}
    for s in samples:
        counts[(s["split"], s["language"])] = counts.get((s["split"], s["language"]), 0) + 1
    print(json.dumps({f"{k[0]}/{k[1]}": v for k, v in sorted(counts.items())}))


if __name__ == "__main__":
    main()
