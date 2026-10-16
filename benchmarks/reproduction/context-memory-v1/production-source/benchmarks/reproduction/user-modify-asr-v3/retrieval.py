"""Inference-only shortlist policies; no transcript replacement or test labels."""
from common import v2_policy


def converted(entries, config):
    return [dict(id=e["id"], source=e["from"], target=e["to"], approved=e["approved"],
                 updatedDay=config["logicalDay"], useCount=e.get("uses",0), recallCount=e.get("recalls",0))
            for e in entries]


def compete(text, entries, config):
    # Multiple actual learned aliases of one destination compete as a group.
    grouped={}
    for e in v2_policy.eligible_entries(entries,config):
        if v2_policy.entity_present(text,e["target"]):
            continue
        for field in ["source","target"]:
            span=v2_policy.best_span(text,e[field])
            if span is None or span[0]>config["competitionThreshold"]:
                continue
            hit=dict(id=e["id"],target=e["target"],score=span[0],start=span[1],width=span[2],
                     phoneticSpan=span[3],matchedField=field,recallCount=e["recallCount"],useCount=e["useCount"])
            old=grouped.get(e["target"])
            rank=lambda h:(h["score"],-h["recallCount"],-h["useCount"],h["id"])
            if old is None or rank(hit)<rank(old):
                grouped[e["target"]]=hit
    candidates=sorted(grouped.values(),key=lambda h:(h["score"],-h["recallCount"],-h["useCount"],h["id"]))
    selected=[]
    for h in candidates:
        # Overlapping text-derived phonetic spans represent competing terms.
        # This is lexical abstention, not the acoustic posterior in PTC-Bias.
        overlaps=lambda g: max(g["start"],h["start"])<min(g["start"]+g["width"],h["start"]+h["width"])
        if any(g["target"]!=h["target"] and overlaps(g) and g["score"]<=h["score"]+config["competitionMargin"] for g in candidates):
            continue
        if any(overlaps(g) for g in selected):
            continue
        selected.append(h)
        if len(selected)==config["retrievalLimit"]:
            break
    return selected


def context_for(arm,text,entries,config):
    pool=converted(entries,config)
    if arm in ["v2_hint","native_vocab"]:
        selected=v2_policy.retrieve(text,pool,config,config["fuzzyThreshold"])
    elif arm=="competing_vocab":
        selected=compete(text,pool,config)
    else:
        raise ValueError("Unknown memory policy "+arm)
    if not selected:
        return "",selected
    words=", ".join(x["target"] for x in selected)
    return ("The relevant terms are "+words+"." if arm=="v2_hint" else "Vocabulary: "+words+"."),selected
