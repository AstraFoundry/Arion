"""Replay all110 actual candidate bodies against reviewed v3 evidence."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
s=importlib.util.spec_from_file_location('work_scope_report_core_v4',HERE.parent/'gguf_work_scope_report.py');core=importlib.util.module_from_spec(s);s.loader.exec_module(core)
study=core.Study(HERE,'rewrite-work-scope-v3');checked=study.checked;collect=study.collect;validate=study.validate;RUN=study.run;read=core.read;rowhash=core.rowhash
if __name__=='__main__':study.main()
