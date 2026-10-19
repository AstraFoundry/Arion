"""Reuse unchanged six-arm runner with an explicitly identified novel corpus."""
import importlib.util
import pathlib

HERE = pathlib.Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('original_chain_runner', HERE.parent / 'correction-chain-v1/run.py')
shared = importlib.util.module_from_spec(spec)
spec.loader.exec_module(shared)
shared.FIXTURES = shared.ROOT / 'benchmarks/corpora/correction-chain-expanded-v2'
original_provenance = shared.provenance

def provenance(config, corpus):
    meta = original_provenance(config, corpus)
    meta['study'] = 'correction-chain-expanded-v2'
    meta['fixtureDirectory'] = str(shared.FIXTURES.relative_to(shared.ROOT))
    for source in HERE.iterdir():
        if source.is_file():
            meta['sourceHashes'][str(source.relative_to(shared.ROOT))] = shared.sha(source)
    rubric = HERE.parent / 'correction-chain-semantic-v1/rubric.json'
    meta['sourceHashes'][str(rubric.relative_to(shared.ROOT))] = shared.sha(rubric)
    return meta

shared.provenance = provenance
if __name__ == '__main__':
    shared.main()
