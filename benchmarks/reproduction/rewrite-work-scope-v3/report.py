"""Replay all94 historical work cases and paired16 challenge controls."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
s=importlib.util.spec_from_file_location('work_scope_report_core_v3',HERE.parent/'gguf_work_scope_report.py');core=importlib.util.module_from_spec(s);s.loader.exec_module(core)
study=core.Study(HERE,'rewrite-history-format-v2');checked=study.checked;collect=study.collect;validate=study.validate;RUN=study.run;read=core.read;rowhash=core.rowhash
if __name__=='__main__':study.main()
