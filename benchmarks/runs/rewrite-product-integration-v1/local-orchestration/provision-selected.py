"""Install the already verified selected artifact into the active local profile."""
import pathlib,json,hashlib,os
root=pathlib.Path('/Users/steven/code/Arion')
catalog=json.loads((root/'models/catalog.json').read_text());model=catalog['rewriters'][0]
name,spec=next(iter(model['files'].items()));source=root/'.workspace/rewrite-small-models-v1'/name
assert source.stat().st_size==spec['sizeBytes']
assert hashlib.file_digest(source.open('rb'),'sha256').hexdigest()==spec['sha256']
profile=pathlib.Path.home()/'Library/Application Support/Arion'
final=profile/'models'/model['id'];assert not final.exists()
staging=final.with_name('.staging-'+model['id']+'-local-adoption');staging.mkdir(mode=0o700)
os.link(source,staging/name)
(staging/'.verified.json').write_text(json.dumps(dict(artifact=model['id'],archiveSha256='',files={name:spec['sha256']})))
(staging/'.verified.json').chmod(0o600);staging.rename(final)
(root/'.workspace/rewrite-product-integration/provisioned.json').write_text(json.dumps(dict(model=model['id'],path=str(final/name),bytes=spec['sizeBytes'],sha256=spec['sha256'],method='verified artifact; hard link within the same local filesystem; original research path retained'),indent=2)+'\n')
print(final)
