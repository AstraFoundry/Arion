"""Compare exact GGUF tokenizer/architecture metadata payloads without tensor decoding."""
import struct,hashlib,pathlib,json
WIDTH={0:1,1:1,2:2,3:2,4:4,5:4,6:4,7:1,10:8,11:8,12:8}
def identities(path):
 with pathlib.Path(path).open('rb') as stream:
  def u32():return struct.unpack('<I',stream.read(4))[0]
  def u64():return struct.unpack('<Q',stream.read(8))[0]
  def skip(kind):
   if kind in WIDTH:stream.seek(WIDTH[kind],1)
   elif kind==8:stream.seek(u64(),1)
   elif kind==9:
    subtype,count=u32(),u64()
    if subtype in WIDTH:stream.seek(WIDTH[subtype]*count,1)
    else:
     for _ in range(count):skip(subtype)
   else:raise ValueError('Unknown GGUF metadata type')
  assert stream.read(4)==b'GGUF';version=u32();tensors=u64();fields=u64();result={}
  for _ in range(fields):
   key=stream.read(u64()).decode();start=stream.tell();kind=u32();skip(kind);end=stream.tell()
   if key.startswith(('tokenizer.','llama.')):
    stream.seek(start);payload=stream.read(end-start);result[key]=dict(type=kind,bytes=len(payload),sha256=hashlib.sha256(payload).hexdigest())
  return dict(version=version,tensorCount=tensors,fields=result)
if __name__=='__main__':
 import sys
 left,right=map(identities,sys.argv[1:3]);assert left==right
 print(json.dumps(dict(identical=True,tokenizerFields=sum(k.startswith('tokenizer.') for k in left['fields']),architectureFields=sum(k.startswith('llama.') for k in left['fields']),identities=left),indent=2))
