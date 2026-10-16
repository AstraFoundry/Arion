"""Record content-free progress and concurrent host processes."""
import json
import subprocess
import time
from collections import Counter
from pathlib import Path

base = Path('.workspace/benchmarks/phonetic-asr-v2')
rows = [json.loads(line) for line in (base / 'correction/results.jsonl').read_text().splitlines()]
rows = [row for row in rows if row['model'] != 'raw']
processes = []
for line in subprocess.check_output(['ps', '-Ao', 'pid,pcpu,rss,comm'], text=True).splitlines()[1:]:
    fields = line.split(None, 3)
    if len(fields) == 4 and ('.workspace/ablation-venv/bin/python' in fields[3] or fields[3].endswith('/product-worker')):
        processes.append(dict(pid=int(fields[0]), cpuPercent=float(fields[1]), rssKiB=int(fields[2]), executable=fields[3]))
with (base / 'host-contention.jsonl').open('a') as stream:
    stream.write(json.dumps(dict(unixSeconds=time.time(), completed=len(rows), processes=processes)) + '\n')
print(json.dumps(dict(completed=len(rows), total=1848, byModel=dict(Counter(row['model'] for row in rows)), truncated=sum(row['response']['truncated'] for row in rows), wrappedText=sum(row['variant'] != 'ipa_only' and row['response']['text'].lstrip().startswith(('{', '[')) for row in rows))))
