"""Synthesizes pause-segmented dictation fixtures for the live pipeline benchmark.

Each case's speech parts are spoken separately and joined with 1.2 s of
silence, longer than the 0.8 s endpoint, so the streaming recognizer closes
a segment at every pause. Usage: build-live-audio.py <text corpus> <out dir>
"""
import hashlib
import json
import pathlib
import re
import subprocess
import sys
import wave

corpus_path, out = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
out.mkdir(parents=True, exist_ok=False)
corpus = json.loads(corpus_path.read_text())
RATE = 16000
PAUSE, LEAD, TRAIL = 1.2, 0.3, 0.8


def speak(text, voice, path):
    aiff = path.with_suffix('.aiff')
    subprocess.run(['say', '-v', voice, '-r', '175', '-o', str(aiff), text], check=True)
    subprocess.run(['ffmpeg', '-loglevel', 'error', '-y', '-i', str(aiff), '-ar', str(RATE), '-ac', '1', '-c:a', 'pcm_s16le', str(path)], check=True)
    aiff.unlink()
    with wave.open(str(path), 'rb') as w:
        frames = w.readframes(w.getnframes())
    path.unlink()
    return frames


def silence(seconds):
    return b'\x00\x00' * int(RATE * seconds)


samples = []
for s in corpus['samples']:
    if s['mode'] == 'verbatim':
        continue
    speech = s.get('speech')
    if speech is None:
        # Development cases have no pause script: pause at commas.
        speech = re.sub(r'[，,]\s*', '|', s['sourceText'].rstrip('。.!?！？'))
    if not speech:
        continue
    parts = [p for p in speech.split('|') if p.strip()]
    voice = 'Samantha' if s['language'] == 'en' else 'Tingting'
    pcm = silence(LEAD)
    for i, part in enumerate(parts):
        if i:
            pcm += silence(PAUSE)
        pcm += speak(part, voice, out / f"{s['id']}-{i}.wav")
    pcm += silence(TRAIL)
    wav = out / f"{s['id']}.wav"
    with wave.open(str(wav), 'wb') as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(RATE)
        w.writeframes(pcm)
    samples.append(dict(id=s['id'], language=s['language'], voice=voice, segments=len(parts), speech='|'.join(parts),
                        audio=dict(path=wav.name, sha256=hashlib.sha256(wav.read_bytes()).hexdigest(), durationMs=len(pcm) / 2 / RATE * 1000)))
manifest = dict(version=1, kind='arion-live-audio', textCorpus=corpus_path.name,
                textCorpusSha256=hashlib.sha256(corpus_path.read_bytes()).hexdigest(),
                synthesis=f'macOS say (Tingting zh/mixed, Samantha en) rate 175; {PAUSE}s pauses between parts; {LEAD}s lead, {TRAIL}s trail; 16 kHz mono PCM16',
                limitations='Synthetic TTS speech with inserted pauses; not natural dictation', samples=samples)
(out / 'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2))
print('Built', len(samples), 'fixtures in', out)
