"""Reuse the actual continuous runner with only the cache-enabled payload adapter."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
prior=module('prefix_chain_runner',HERE.parent/'rewrite-metal-chain-v1/run.py')
prior.HERE=HERE;prior.pipeline=module('prefix_payloads',HERE/'pipeline.py')
if __name__=='__main__':prior.main()
