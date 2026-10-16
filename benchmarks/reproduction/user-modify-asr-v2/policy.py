"""Retrieve approved spellings from first-pass text for audio re-decoding.

Source spellings are retrieval keys only. They never enter the audio model's
context in the optimized treatments. Text-derived pinyin is not new acoustics.
"""
import importlib.util
from pathlib import Path
import re
import unicodedata
from pypinyin import lazy_pinyin, Style

V1_PATH = Path(__file__).resolve().parents[1]/'user-modify-asr-v1/run.py'
spec = importlib.util.spec_from_file_location('memory_v1', V1_PATH)
v1 = importlib.util.module_from_spec(spec)
spec.loader.exec_module(v1)
sha = v1.sha


def entity_present(text, term):
    text = unicodedata.normalize('NFKC', text).lower()
    term = unicodedata.normalize('NFKC', term).lower()
    if re.search(r'[\u3400-\u9fff]', term):
        return term in text
    letters = re.sub(r'\s+', '', term)
    pattern = r'\s*'.join(re.escape(c) for c in letters)
    return re.search(r'(?<![a-z0-9])'+pattern+r'(?![a-z0-9])', text) is not None


def eligible_entries(entries, config):
    pairs={(e['source'],e['target']) for e in entries if e['approved']}
    return [e for e in entries if e['approved']
            and config['logicalDay']-e['updatedDay']<config['expireDays']
            and (e['target'],e['source']) not in pairs]


def distance(a,b):
    """Use the same Levenshtein recurrence as the benchmark metric."""
    import editdistance
    return editdistance.eval(a,b)


def phonetic_tokens(text):
    # Punctuation delimits possible spans; Latin words remain lexical tokens.
    chunks = re.findall(r'[\u3400-\u9fff]+|[a-zA-Z0-9]+',text.lower())
    result=[]
    for chunk in chunks:
        result.extend(lazy_pinyin(chunk,style=Style.NORMAL,errors=lambda value:[value]))
    return result


def best_span(text, source):
    query, key=phonetic_tokens(text),phonetic_tokens(source)
    if not key or not query:return None
    n=len(key);best=None
    for width in range(max(1,n-1),n+2):
        for start in range(len(query)-width+1):
            span=query[start:start+width]
            # Chinese syllables compare at letter level to allow kou/ku;
            # Latin multi-word keys compare without whitespace as well.
            flat_key=''.join(key);flat_span=''.join(span)
            score=distance(flat_key,flat_span)/max(len(flat_key),len(flat_span))
            candidate=(score,start,width,' '.join(span))
            if best is None or candidate<best:best=candidate
    return best


def retrieve(text,entries,config,threshold):
    hits=[];seen=set()
    for e in eligible_entries(entries,config):
        if entity_present(text,e['target']):
            continue
        span=best_span(text,e['source'])
        if span is None or span[0]>threshold:continue
        hits.append(dict(id=e['id'],target=e['target'],score=span[0],start=span[1],width=span[2],phoneticSpan=span[3],recallCount=e['recallCount'],useCount=e['useCount']))
    hits.sort(key=lambda h:(h['score'],-h['recallCount'],-h['useCount'],h['id']))
    selected=[]
    for h in hits:
        if h['target'] not in seen:
            selected.append(h);seen.add(h['target'])
        if len(selected)==config['retrievalLimit']:break
    return selected


def context_for(arm,text,entries,config):
    if arm=='legacy_pairs':
        selected=v1.select(entries,config)
        return '\n'.join(e['source']+' → '+e['target'] for e in selected),[dict(id=e['id'],target=e['target']) for e in selected]
    if arm=='global_destinations':
        selected=eligible_entries(entries,config)
        return ' '.join(e['target'] for e in selected),[dict(id=e['id'],target=e['target']) for e in selected]
    if arm not in ['retrieved_exact','retrieved_fuzzy','retrieved_hint']:
        raise ValueError('Unknown arm '+arm)
    threshold=0 if arm=='retrieved_exact' else config['fuzzyThreshold']
    selected=retrieve(text,entries,config,threshold)
    terms=[h['target'] for h in selected]
    if not terms:return '',selected
    if arm=='retrieved_hint':return 'The relevant terms are '+', '.join(terms)+'.',selected
    return ' '.join(terms),selected
