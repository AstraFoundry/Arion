# Packaged speech smoke inputs

These are the unchanged public development fixtures used by the Windows packaged
speech test. Their original path, byte identity and attribution are recorded in
the [public fixture manifest](../../../../scripts/testdata/public-fixtures.json).

`libri-251-137823-0017.wav` is a LibriSpeech utterance from speaker 251, chapter
137823, retained from the recorded `openslr/librispeech_asr@71cacbfb` corpus. The
LibriSpeech source is [OpenSLR SLR12](https://www.openslr.org/12), distributed under
[CC BY 4.0](https://creativecommons.org/licenses/by/4.0/). The copied archived WAV
bytes are unchanged. `nonspeech-dev-0.wav` is the original generated nonspeech
fixture.

The test still requires the packaged app, an explicitly provisioned real model
and Windows owned controls. Retaining these inputs does not establish successful
Windows execution or general speech quality.
