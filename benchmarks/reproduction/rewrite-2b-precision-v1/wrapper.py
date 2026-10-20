"""Keep every Q4 baseline prompt/encoding byte unchanged for the Q8 artifact screen."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('precision_private_wrapper',HERE.parent/'rewrite-private-spans-v1/wrapper.py');prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior);prior.HERE=HERE
def payload(sample,arm):return prior.payload(sample,'qwen-private-spans')
def decode(raw,encoding):return prior.decode(raw,encoding)
