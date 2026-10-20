#!/usr/bin/env python3
"""Hash-bound stage report. Automatic checks are diagnostic proxies only."""
import argparse
import hashlib
import html
import json
import math
import re
import shutil
from pathlib import Path


def digest(data):
    return hashlib.sha256(data).hexdigest()


def canonical(value):
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(',', ':')).encode()


def load_json(path):
    return json.loads(Path(path).read_bytes())


def load_arm(path, case_ids):
    raw = Path(path).read_bytes()
    rows = [json.loads(line) for line in raw.splitlines() if line.strip()]
    if not rows or rows[0].get('kind') != 'start' or rows[-1].get('kind') != 'complete':
        raise ValueError(f'incomplete arm: {path}; retain failed attempt, do not report as complete')
    observations = {}
    for row in rows:
        if row.get('kind') == 'observation':
            key = (row['id'], row['phase'])
            if key in observations:
                raise ValueError(f'duplicate observation: {path} {key}')
            observations[key] = row
        elif row.get('kind') in ('load-error', 'session-error'):
            raise ValueError(f'failed arm: {path}; retain failed attempt separately')
    expected = {(i, phase) for i in case_ids for phase in ('preview', 'final')}
    if set(observations) != expected:
        raise ValueError(f'coverage mismatch: {path}; missing={sorted(expected-set(observations))}, extra={sorted(set(observations)-expected)}')
    return {'sha256': digest(raw), 'observations': observations, 'events': rows}


def stage_details(observation):
    stages = []
    for call in observation.get('wire') or []:
        request = call.get('request') or {}
        response = call.get('response') or {}
        if not isinstance(request, dict) or not isinstance(response, dict):
            raise ValueError('wire request/response must retain parsed native JSON')
        messages = request.get('messages') or []
        source = None
        user = next((m for m in reversed(messages) if m.get('role') == 'user'), {})
        try:
            payload = json.loads(user.get('content', '{}'))
            if isinstance(payload, dict):
                source = payload.get('source')
        except (json.JSONDecodeError, TypeError):
            payload = None
        is_gaps = isinstance(payload, dict) and 'parts' in payload
        kind = 'selector' if isinstance(payload, dict) and 'hypotheses' in payload else 'formatter'
        choices = response.get('choices') or []
        text = choices[0].get('message', {}).get('content') if choices else None
        rendered = text
        rendering_error = ''
        if is_gaps:
            try:
                gaps = json.loads(text)
                ids = [part['gap'] for part in payload['parts'] if 'gap' in part]
                if not isinstance(gaps, dict) or set(gaps) != set(ids) or any(not isinstance(v, str) for v in gaps.values()):
                    raise ValueError('gap response does not cover exact gap IDs')
                rendered = ''.join(gaps[part['gap']] if 'gap' in part else part['text'] for part in payload['parts'])
            except (json.JSONDecodeError, TypeError, KeyError, ValueError) as exc:
                rendered, rendering_error = None, str(exc)
        stages.append({'kind': kind, 'protocol': 'fixed-gaps' if is_gaps else 'plain',
                       'source': source, 'rawOutput': text, 'renderedProposal': rendered,
                       'renderingError': rendering_error, 'elapsedMs': call.get('elapsedMs'),
                       'status': call.get('status'), 'error': call.get('error', ''), 'request': request, 'response': response})
    return stages


def quantiles(values):
    values = sorted(v for v in values if isinstance(v, (int, float)) and math.isfinite(v))
    if not values:
        return {'count': 0, 'p50': None, 'p95': None}
    # Nearest rank; repeats/cases are not independent speakers.
    return {'count': len(values), 'p50': values[max(0, math.ceil(.5 * len(values)) - 1)],
            'p95': values[max(0, math.ceil(.95 * len(values)) - 1)]}


def review_template(cases_path, arms):
    records = []
    for name, arm in arms.items():
        for (case_id, phase), observation in arm['observations'].items():
            records.append({'arm': name, 'id': case_id, 'phase': phase,
                            'observationSha256': digest(canonical(observation)),
                            'fidelity': 'unreviewed', 'note': ''})
    return {'schema': 'rewrite-stage-review-v1', 'corpusSha256': digest(Path(cases_path).read_bytes()),
            'arms': {name: arm['sha256'] for name, arm in arms.items()}, 'records': records}


def validate_review(review, template):
    for key in ('schema', 'corpusSha256', 'arms'):
        if review.get(key) != template[key]:
            raise ValueError(f'review {key} is not bound to these exact inputs')
    expected = {(r['arm'], r['id'], r['phase']): r for r in template['records']}
    actual = {}
    for row in review.get('records', []):
        key = (row['arm'], row['id'], row['phase'])
        if key in actual or key not in expected:
            raise ValueError(f'duplicate or unexpected manual review record: {key}')
        if row.get('observationSha256') != expected[key]['observationSha256']:
            raise ValueError(f'manual review observation hash mismatch: {key}')
        if row.get('fidelity') not in ('correct', 'incorrect', 'unreviewed', 'not-assessable'):
            raise ValueError(f'unsupported manual review verdict: {key}')
        if row['fidelity'] != 'unreviewed' and not str(row.get('note', '')).strip():
            raise ValueError(f'reviewed verdict requires a rationale: {key}')
        actual[key] = row
    if set(actual) != set(expected):
        raise ValueError('manual review coverage mismatch; retain unreviewed entries explicitly')
    return actual


def build_report(cases_path, arms, review):
    cases = load_json(cases_path)
    template = review_template(cases_path, arms)
    reviews = validate_review(review, template)
    summaries = {}
    report_cases = []
    for case in cases:
        row = {'fixture': case, 'arms': {}}
        terms = case.get('checks') or []
        entity = terms[0] if terms else None
        for name, arm in arms.items():
            phases = {}
            for phase in ('preview', 'final'):
                observation = arm['observations'][(case['id'], phase)]
                stages = stage_details(observation)
                formatter = next((s for s in reversed(stages) if s['kind'] == 'formatter'), None)
                output = observation.get('result', {}).get('text', '')
                fmt_source = formatter['source'] if formatter else None
                raw_output = formatter['renderedProposal'] if formatter else None
                # Only a completed formatter output can establish this text proxy.
                complete_formatter = bool(formatter and formatter.get('status') == 200 and
                    not formatter.get('error') and formatter['response'].get('choices') and
                    formatter['response']['choices'][0].get('finish_reason') == 'stop')
                phases[phase] = {'observation': observation, 'stages': stages,
                    'manualReview': reviews[(name, case['id'], phase)],
                    'entityPresent': entity in output if entity else None,
                    'selectedTermDamageProxy': bool(complete_formatter and entity and isinstance(fmt_source, str) and
                         entity in fmt_source and isinstance(raw_output, str) and fmt_source.count(entity) != raw_output.count(entity)),
                    'constraintContains': {term: term in output for term in terms[1:]},
                    'evaluatorPhaseChecks': {
                        'contains': {term: term in output for term in case.get('phaseChecks', {}).get(phase, {}).get('contains', [])},
                        'excludes': {term: term not in output for term in case.get('excludes', []) + case.get('phaseChecks', {}).get(phase, {}).get('excludes', [])},
                        'counts': {term: {'expected': count, 'actual': output.count(term)}
                                   for term, count in case.get('phaseChecks', {}).get(phase, {}).get('counts', {}).items()}},
                    'failed': bool(observation.get('error') or observation.get('result', {}).get('cancelled') or
                                   observation.get('result', {}).get('truncated') or not output)}
            row['arms'][name] = {'phases': phases, 'previewFinalRegressionProxy':
                 bool(entity and phases['preview']['entityPresent'] and not phases['final']['entityPresent']),
                 'entityCheck': entity}
        report_cases.append(row)
    for name, arm in arms.items():
        observations = [phase for row in report_cases for phase in row['arms'][name]['phases'].values()]
        stage_times = {kind: quantiles([stage.get('elapsedMs') for o in observations for stage in o['stages'] if stage['kind'] == kind])
                       for kind in ('selector', 'formatter')}
        summaries[name] = {'sha256': arm['sha256'], 'pairDenominator': len(cases), 'phaseDenominator': len(observations),
            'failedPhases': sum(o['failed'] for o in observations),
            'entityEvaluablePairs': sum(bool(row['arms'][name]['entityCheck']) for row in report_cases),
            'previewEntityCorrectPairs': sum(bool(row['arms'][name]['entityCheck'] and row['arms'][name]['phases']['preview']['entityPresent']) for row in report_cases),
            'previewFinalRegressionProxy': sum(row['arms'][name]['previewFinalRegressionProxy'] for row in report_cases),
            'selectedTermDamageProxy': sum(o['selectedTermDamageProxy'] for o in observations),
            'manualReviewCounts': {status: sum(o['manualReview']['fidelity'] == status for o in observations)
                                   for status in ('correct', 'incorrect', 'unreviewed', 'not-assessable')},
            'latencyMs': {'phases': quantiles([o['observation'].get('elapsedMs') for o in observations]), **stage_times},
            'events': arm['events']}
    return {'schema': 'rewrite-stage-report-v1', 'corpusSha256': template['corpusSha256'],
            'reviewer': review.get('reviewer', {'kind': 'unspecified', 'limitation': 'Reviewer identity/independence not declared; do not assume human or independent review.'}),
            'limitations': ['Substring counts are diagnostic proxies, not semantic quality verdicts.',
                'Constructed transcripts do not demonstrate acoustic or OS OCR execution.',
                'Native ASR/Vision cases use synthetic voice/screen fixtures, not the user\'s original incident.',
                'Discovery and tuned regressions are not held out; no significance or release-quality claim follows.',
                'All phases remain in denominators, including failed/unreviewed observations.',
                'Selected-term damage refers to expected term present in formatter input and its occurrence count changed in the completed proposal; fixed-gap responses are deterministically stitched before this check. It does not prove correctness of selector intent.',
                'Preview→final proxy requires a target-present preview; inspect first failing stage and source/context differences.'],
            'arms': summaries, 'cases': report_cases}


HTML = r'''<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Rewrite stage evidence</title>
<style>body{margin:0;background:#f6f5f1;color:#222;font:15px/1.5 system-ui}main{max-width:1800px;margin:auto;padding:30px}h1{font-size:26px;margin-bottom:8px}h2{font-size:20px}h3{font-size:17px}p{max-width:1100px}.muted{color:#65645f}select,input{font:inherit;padding:8px;margin-right:8px;border:1px solid #aaa;border-radius:5px;background:white}table{border-collapse:collapse;width:100%;background:white}th,td{padding:9px;text-align:left;border:1px solid #ddd;vertical-align:top}pre{white-space:pre-wrap;overflow-wrap:anywhere;background:#f4f3ef;padding:12px;border-radius:4px;font:13px/1.55 ui-monospace,monospace;margin:6px 0 14px}details{margin:10px 0}summary{cursor:pointer}.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(360px,1fr));gap:16px;align-items:start}.card{background:white;border:1px solid #d8d6ce;border-radius:8px;padding:18px;min-width:0}.bad{color:#a12f26}.good{color:#236442}.tag{display:inline-block;font-size:12px;background:#ebe9e1;padding:3px 7px;border-radius:4px}label{display:block;margin:16px 0 10px}button{font:inherit;padding:7px 12px;margin-right:8px;cursor:pointer}.phase{border-top:1px solid #ddd;margin-top:20px;padding-top:6px}</style>
<main><h1>Rewrite stage evidence</h1><p class="muted">Exact native stage traces, paired arms, and hash-bound manual review. Counts below are text proxies; no automatic semantic pass is inferred.</p><div id="summary"></div><details><summary>Evidence limitations</summary><ul id="limits"></ul></details><label>Case <select id="cases"></select><input id="search" placeholder="Search source, OCR, ID"><select id="filter"><option value="all">All cases</option><option value="regression">Preview → final regression proxy</option><option value="damage">Formatter term damage proxy</option><option value="error">Errors</option><option value="unreviewed">Unreviewed</option></select></label><div id="count"></div><div id="fixture"></div><div id="cards" class="cards"></div></main>
<script id="data" type="application/json">__DATA__</script><script>
const data=JSON.parse(document.getElementById('data').textContent),el=id=>document.getElementById(id),esc=s=>String(s??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c])),pretty=v=>JSON.stringify(v,null,2),pre=v=>'<pre>'+esc(typeof v==='string'?v:pretty(v))+'</pre>';
const names=Object.keys(data.arms);el('limits').innerHTML=data.limitations.map(x=>'<li>'+esc(x)+'</li>').join('');
el('summary').innerHTML='<details><summary>Reviewer provenance and independence</summary>'+pre(data.reviewer)+'</details><table><thead><tr><th>Arm</th><th>Pairs / phases / failures</th><th>Preview target present → lost</th><th>Raw formatter damage</th><th>Manual review counts</th><th>Latency ms (p50 / p95)</th></tr></thead><tbody>'+names.map(n=>{let a=data.arms[n];return '<tr><td>'+esc(n)+'</td><td>'+a.pairDenominator+' / '+a.phaseDenominator+' / '+a.failedPhases+'</td><td>'+a.previewFinalRegressionProxy+' / '+a.previewEntityCorrectPairs+' eligible previews ('+a.entityEvaluablePairs+' entity pairs)</td><td>'+a.selectedTermDamageProxy+' / '+a.phaseDenominator+' phases</td><td>'+esc(pretty(a.manualReviewCounts))+'</td><td>'+Object.entries(a.latencyMs).map(([k,v])=>esc(k)+': '+v.p50+' / '+v.p95+' (n='+v.count+')').join('<br>')+'</td></tr>'}).join('')+'</tbody></table>';
let shown=[];function refresh(){let old=el('cases').value,q=el('search').value.toLowerCase(),f=el('filter').value;shown=data.cases.filter(c=>{let arms=Object.values(c.arms),ph=arms.flatMap(a=>Object.values(a.phases));return pretty(c.fixture).toLowerCase().includes(q)&&(f==='all'||f==='regression'&&arms.some(a=>a.previewFinalRegressionProxy)||f==='damage'&&ph.some(p=>p.selectedTermDamageProxy)||f==='error'&&ph.some(p=>p.failed)||f==='unreviewed'&&ph.some(p=>p.manualReview.fidelity==='unreviewed'))});el('cases').innerHTML=shown.map(c=>'<option value="'+esc(c.fixture.id)+'">'+esc(c.fixture.id)+'</option>').join('');if(shown.some(c=>c.fixture.id===old))el('cases').value=old;el('count').textContent=shown.length+' / '+data.cases.length+' cases';render()}
function render(){let c=shown.find(c=>c.fixture.id===el('cases').value);if(!c){el('cards').innerHTML='';el('fixture').innerHTML='';return}let x=c.fixture;el('fixture').innerHTML='<h2>'+esc(x.id)+' <span class="tag">'+esc(x.pairKind||'origin unspecified')+'</span></h2><details><summary>Frozen fixture, OCR/context, and evaluator checks</summary>'+pre(x)+'</details>';el('cards').innerHTML=names.map(name=>{let arm=c.arms[name];return '<article class="card"><h2>'+esc(name)+'</h2><p class="'+(arm.previewFinalRegressionProxy?'bad':'muted')+'">Preview → final target-loss proxy: '+arm.previewFinalRegressionProxy+'</p>'+Object.entries(arm.phases).map(([phase,p])=>{let o=p.observation,r=o.result||{};return '<section class="phase"><h3>'+esc(phase)+' · '+esc(o.mode)+'</h3><p>'+esc(p.manualReview.fidelity)+' · '+esc(p.manualReview.note)+'</p><p class="'+(p.failed?'bad':'muted')+'">Error: '+esc(o.error||'none')+'; delivery: '+esc(o.deliveryVerdict)+'; entity present: '+p.entityPresent+'; raw formatter damage proxy: '+p.selectedTermDamageProxy+'</p><b>Source</b>'+pre(o.request.Text??o.request.text)+'<b>Delivered result</b>'+pre(r.text||'')+p.stages.map(s=>'<details open><summary>'+esc(s.kind)+' · '+s.elapsedMs+' ms</summary><b>Stage input source</b>'+pre(s.source)+'<b>Native raw output</b>'+pre(s.rawOutput)+(s.protocol==='fixed-gaps'?'<b>Deterministically stitched native proposal</b>'+pre(s.renderedProposal):'')+'<details><summary>Exact request / response</summary>'+pre({request:s.request,response:s.response,status:s.status,error:s.error})+'</details></details>').join('')+'<details><summary>Complete phase observation and hash-bound review</summary>'+pre(p)+'</details></section>'}).join('')+'</article>'}).join('')}
el('cases').onchange=render;el('search').oninput=refresh;el('filter').onchange=refresh;refresh();</script></html>'''


def verify_archive(directory):
    directory = Path(directory).resolve()
    manifest = load_json(directory / 'manifest.json')
    for relative, expected in manifest['files'].items():
        path = (directory / relative).resolve()
        if not path.is_relative_to(directory) or not path.is_file() or digest(path.read_bytes()) != expected:
            raise ValueError(f'archive hash mismatch or missing file: {relative}')
    actual = {str(p.relative_to(directory)) for p in directory.rglob('*') if p.is_file()} - {'manifest.json'}
    if actual != set(manifest['files']):
        raise ValueError('archive file inventory mismatch')
    return manifest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--cases', type=Path)
    parser.add_argument('--arm', action='append', default=[], help='NAME=JSONL; repeated for baseline/candidate/ablations')
    parser.add_argument('--review', type=Path, help='Mandatory hash-bound manual review; unreviewed records remain explicit')
    parser.add_argument('--init-review', type=Path, help='Write exclusive unreviewed template, then exit')
    parser.add_argument('--out', type=Path, help='New report archive directory; never overwritten')
    parser.add_argument('--provenance', type=Path, help='Optional artifact/hardware/runtime provenance JSON copied verbatim')
    parser.add_argument('--verify', type=Path, help='Verify an existing report archive')
    args = parser.parse_args()
    if args.verify:
        verify_archive(args.verify)
        print('Verified archive:', args.verify.resolve())
        return
    if not args.cases or not args.arm:
        parser.error('--cases and at least one --arm are required')
    cases = load_json(args.cases)
    ids = [c['id'] for c in cases]
    if not ids or len(set(ids)) != len(ids):
        raise ValueError('empty corpus or duplicate case ID')
    arms, paths = {}, {}
    for spec in args.arm:
        name, path = spec.split('=', 1)
        if not re.fullmatch('[a-zA-Z0-9_-]+', name) or name in arms:
            raise ValueError('arm names must be unique safe identifiers')
        arms[name], paths[name] = load_arm(path, ids), Path(path)
    if args.init_review:
        with args.init_review.open('x') as handle:
            json.dump(review_template(args.cases, arms), handle, ensure_ascii=False, indent=2)
            handle.write('\n')
        print('Unreviewed template:', args.init_review.resolve())
        return
    if not args.review or not args.out:
        parser.error('--review and --out are required; initialize review with --init-review')
    report = build_report(args.cases, arms, load_json(args.review))
    args.out.mkdir(parents=True, exist_ok=False)
    sources = args.out / 'sources'
    sources.mkdir()
    shutil.copyfile(args.cases, sources / 'cases.json')
    shutil.copyfile(args.review, sources / 'manual-review.json')
    shutil.copyfile(Path(__file__), sources / 'report.py')
    for name, path in paths.items():
        shutil.copyfile(path, sources / f'{name}.jsonl')
    if args.provenance:
        shutil.copyfile(args.provenance, sources / 'provenance.json')
    (args.out / 'report.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    payload = json.dumps(report, ensure_ascii=False).replace('<', '\\u003c')
    (args.out / 'index.html').write_text(HTML.replace('__DATA__', payload))
    files = {str(p.relative_to(args.out)): digest(p.read_bytes()) for p in sorted(args.out.rglob('*')) if p.is_file()}
    (args.out / 'manifest.json').write_text(json.dumps({'schema': 'rewrite-stage-archive-v1', 'files': files}, indent=2) + '\n')
    verify_archive(args.out)
    print('Report:', (args.out / 'index.html').resolve())


if __name__ == '__main__':
    main()
