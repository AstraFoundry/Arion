"""Render exact authored reference text as deterministic OCR fixture images."""
import hashlib
import json
import pathlib
import platform
from PIL import Image,ImageDraw,ImageFont
ROOT=pathlib.Path(__file__).resolve().parents[3]
CORPUS=ROOT/'benchmarks/corpora/entity-ocr-v1'
FONT=pathlib.Path('/System/Library/Fonts/STHeiti Medium.ttc')
def main():
    data=json.loads((CORPUS/'corpus.json').read_text());font=ImageFont.truetype(str(FONT),32)
    for sample in data['samples']:
        path=CORPUS/sample['contextFixture']['image']
        if path.exists():raise ValueError('Existing fixture: '+str(path))
        image=Image.new('RGB',(1200,500),'#fbfbf9');draw=ImageDraw.Draw(image)
        for index,line in enumerate(sample['contextFixture']['screenTruth'].splitlines()):
            draw.text((56,70+index*76),line,font=font,fill='#20282a')
        image.save(path)
    provenance=dict(kind='authored-text-images',purpose='Actual native Vision OCR diagnostic, not real window capture.',fontPath=str(FONT),fontSha256=hashlib.sha256(FONT.read_bytes()).hexdigest(),pillowVersion=Image.__version__,pythonVersion=platform.python_version(),images={s['contextFixture']['image']:hashlib.sha256((CORPUS/s['contextFixture']['image']).read_bytes()).hexdigest() for s in data['samples']})
    (CORPUS/'provenance.json').write_text(json.dumps(provenance,indent=2)+'\n')
if __name__=='__main__':main()
