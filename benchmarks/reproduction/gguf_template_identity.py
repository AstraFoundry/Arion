"""Require explicit frozen prompt ownership before future model requests."""
import pathlib,hashlib,json

def identity(root,freeze,pipeline,arm,stage,scene,request):
 declared=pipeline.template_identity(arm,stage,scene)
 if set(declared)!={'version','path'} or not isinstance(declared['version'],str) or not declared['version']:raise ValueError('Missing explicit template identity')
 path=pathlib.Path(declared['path'])
 if path.is_absolute() or '..' in path.parts:raise ValueError('Template must be repository-relative')
 key=path.as_posix();digest=hashlib.sha256((root/path).read_bytes()).hexdigest()
 if freeze['files'].get(key)!=digest:raise ValueError('Template not pinned by pre-inference freeze')
 return dict(templateVersion=declared['version'],templatePath=key,templateSha256=digest,actualMessagesSha256=hashlib.sha256(json.dumps(request['messages'],ensure_ascii=False,sort_keys=True).encode()).hexdigest())
