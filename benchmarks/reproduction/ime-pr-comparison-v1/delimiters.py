"""Compare unchanged first pages for normal Pinyin with explicit boundaries."""
import argparse, json, pathlib
from run import load_variants, observation, read

p = argparse.ArgumentParser()
p.add_argument('work')
p.add_argument('out')
p.add_argument('--variants')
a = p.parse_args()
work, out = pathlib.Path(a.work).resolve(), pathlib.Path(a.out).resolve()
out.mkdir(parents=True, exist_ok=False)
variants = load_variants(a.variants)
results = []
for case, raw in enumerate(["ni'hao", "xi'an"]):
    for page in [5, 9]:
        pages = {}
        for name in variants:
            item = observation(work, out, {'id': str(case), 'input': raw}, name, 1, page, 0, variants)
            assert item['status'] == 'completed', item
            pages[name] = read(out / item['path'])['initial']['state']['candidates']
        assert len({json.dumps(value, ensure_ascii=False) for value in pages.values()}) == 1, pages
        results.append({'input': raw, 'pageSize': page, 'pages': pages})
(out / 'report.json').write_text(json.dumps({'status': 'passed', 'controls': results,
    'boundary': 'Supplemental first-page preservation controls; excluded from frozen corpus and timing.'}, ensure_ascii=False, indent=2) + '\n')
print('Passed four paired normal-delimiter page controls')
