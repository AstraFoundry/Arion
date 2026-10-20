"""Reuse checked paired reporting with Q8-specific input identity and scope."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('precision_pair_report',HERE.parent/'rewrite-data-object-v1/report.py');analysis=importlib.util.module_from_spec(spec);spec.loader.exec_module(analysis)
spec=importlib.util.spec_from_file_location('precision_evidence_driver',HERE/'run.py');driver=importlib.util.module_from_spec(spec);spec.loader.exec_module(driver)
analysis.HERE=HERE;analysis.RUN=analysis.ROOT/'benchmarks/runs/rewrite-2b-precision-v1';analysis.ARMS=list(driver.runner.ARMS);analysis.prior.HERE=HERE;analysis.prior.base.HERE=HERE;analysis.prior.base.runner=driver.runner
analysis.HEADER='# Complete Q8 quantized-artifact review'
analysis.REVIEW_SCOPE='All36 new Q8 native/restored outputs personally inspected, alongside the already-inspected historical36 Q4 baseline. Reused Q4 observations count only in their original round. All36 requests and encodings remain identical; weights/artifact precision differ. Exposed Chinese synthetic development, not independent, natural, minimum16GB or native acceptance. No output rejection or source fallback. Numeric base-weight/converter lineage is not independently proven, so causal quantization effects are not claimed.'
ROOT=analysis.ROOT;RUN=analysis.RUN;ARMS=analysis.ARMS;checked=analysis.checked;baseline=analysis.baseline
if __name__=='__main__':analysis.main()
