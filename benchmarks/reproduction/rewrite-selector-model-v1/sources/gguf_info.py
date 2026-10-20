import struct,pathlib,json,math,sys
TYPES={0:'B',1:'b',2:'H',3:'h',4:'I',5:'i',6:'f',7:'?',10:'Q',11:'q',12:'d'}
def inspect(path):
 with open(path,'rb') as f:
  def read(fmt):return struct.unpack('<'+fmt,f.read(struct.calcsize('<'+fmt)))[0]
  def string():return f.read(read('Q')).decode('utf-8')
  def value(t):
   if t==8:return string()
   if t==9:
    subtype=read('I');n=read('Q');return [value(subtype) for _ in range(n)]
   return read(TYPES[t])
  if f.read(4)!=b'GGUF':raise ValueError('Not GGUF')
  version=read('I');count=read('Q');fields=read('Q');metadata={}
  for _ in range(fields):
   k=string();metadata[k]=value(read('I'))
  tensors=[]
  for _ in range(count):
   name=string();dims=[read('Q') for _ in range(read('I'))];kind=read('I');offset=read('Q');tensors.append(dict(name=name,shape=dims,type=kind,elements=math.prod(dims)))
  return dict(version=version,tensorCount=count,parameters=sum(x['elements'] for x in tensors),metadata={k:v for k,v in metadata.items() if not isinstance(v,list)},tensors=tensors)
if __name__=='__main__':
 d=inspect(sys.argv[1]);print(json.dumps(d,ensure_ascii=False,indent=2))
