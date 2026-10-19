"""Freeze novel authored scripts before synthesizing paired six-arm audio."""
import importlib.util
import json
import pathlib
import wave

ROOT = pathlib.Path(__file__).resolve().parents[3]
HERE = pathlib.Path(__file__).resolve().parent
OUT = ROOT / 'benchmarks/corpora/correction-chain-expanded-v2'
spec = importlib.util.spec_from_file_location('original_fixtures', HERE.parent / 'correction-chain-v1/build_fixtures.py')
shared = importlib.util.module_from_spec(spec)
spec.loader.exec_module(shared)

def main():
    cases = json.loads((HERE / 'cases.json').read_text())
    assert set(cases) == {'zh', 'en', 'mixed'} and all(len(rows) == 32 for rows in cases.values())
    OUT.mkdir(parents=True, exist_ok=False)
    shared.write_json(OUT / 'design.json', dict(cases=cases, casesSha256=shared.sha(HERE / 'cases.json'),
        frozenRubricSha256=shared.sha(HERE.parent / 'correction-chain-semantic-v1/rubric.json'),
        authorClusters=1, status='scripts and rubric frozen before inference',
        limitations=['Synthetic single-author scripts; translated long-form themes are related', 'Two test voices per language, no spontaneous speech or independent human review']))
    samples, labels = [], {}
    for language, rows in cases.items():
        for index, (task, speech, intent) in enumerate(rows):
            identity = f'expanded-{language}-{index:02d}'
            voice = (['Samantha', 'Daniel'] if language == 'en' else ['Tingting', 'Eddy (中文（中国大陆）)'])[index % 2]
            audio = OUT / f'{identity}.wav'
            duration = shared.synthesize(speech, voice, audio)
            samples.append(dict(id=identity, split='test', language=language, task=task,
                scene={'email':'email','list':'list'}.get(task, 'chat'), speakerId=voice,
                audio=dict(path=audio.name, sha256=shared.sha(audio), durationMs=duration)))
            labels[identity] = dict(transcript=speech, references=[intent], required=[], forbidden=[])
            print(identity, round(duration), flush=True)
    for index, (language, task, speech, intent, required, forbidden) in enumerate(shared.DEV):
        identity = f'dev-{language}-{index:02d}'
        voice = 'Flo (英语（美国）)' if language == 'en' else 'Flo (中文（中国大陆）)'
        audio = OUT / f'{identity}.wav'
        duration = shared.synthesize(speech, voice, audio)
        samples.append(dict(id=identity, split='dev', language=language, task=task, scene='chat', speakerId=voice,
            audio=dict(path=audio.name, sha256=shared.sha(audio), durationMs=duration)))
        labels[identity] = dict(transcript=speech, references=[intent], required=required, forbidden=forbidden)
    for index in range(4):
        identity = f'expanded-silence-{index}'
        audio = OUT / f'{identity}.wav'
        frames = 16000 * (index + 1)
        with wave.open(str(audio), 'wb') as target:
            target.setparams((1, 2, 16000, 0, 'NONE', 'not compressed'))
            target.writeframes(b'\0\0' * frames)
        samples.append(dict(id=identity, split='test', language='zh', task='silence', scene='chat', speakerId='generated-silence',
            audio=dict(path=audio.name, sha256=shared.sha(audio), durationMs=frames/16)))
        labels[identity] = dict(transcript='', references=[''], required=[], forbidden=[])
    shared.write_json(OUT / 'inputs.json', dict(version=1, kind='correction-chain-inputs-v1', samples=samples))
    shared.write_json(OUT / 'labels.json', labels)
    shared.write_json(OUT / 'manifest.json', dict(kind='correction-chain-expanded-fixtures-v2',
        license='Developer-authored text; system-voice audio for local research reproduction; no voice/model redistribution',
        files={p.name:shared.sha(p) for p in OUT.iterdir() if p.is_file()}))

if __name__ == '__main__':
    main()
