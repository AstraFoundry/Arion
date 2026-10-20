"""Place trusted editing rules in system scope; keep the editable user tail data-only."""
import importlib.util,json,pathlib
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('system_probe_prior_wrapper',HERE.parent/'rewrite-current-task-v1/wrapper.py');prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior)
RULES=prior.RULES.replace('只编辑上面的待编辑口述','只编辑本轮最后一条 user 消息中 Dictated message to edit 标签之后、直到该消息结尾的待编辑口述')
def payload(sample,arm):
 template=json.loads((HERE/'templates'/(sample['scene']+'.json')).read_text());messages,encoding=prior.codec.messages(sample,template,True)
 messages[0]['content']=messages[0]['content'].replace(prior.codec.POLICY,prior.FLEX_POLICY)
 messages[0]['content']+='\n\n当前编辑任务（可信系统规则）：\n'+prior.TASKS[sample['scene']]+'\n'+RULES
 encoding=dict(encoding,deliveryPolicy='unfiltered')
 return dict(messages=messages,temperature=0,seed=42,max_tokens=384,cache_prompt=False,stream=False),encoding

def decode(raw,encoding):return prior.decode(raw,encoding)
