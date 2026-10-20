"""Balanced before/after data from exact unverified UTF-8 candidate substitutions."""
def augment(original_data,prepared):
 result=original_data(prepared);source=prepared['source'];raw=source.encode()
 if len(result['hypotheses'])!=len(prepared['hypotheses']):raise ValueError('Changed candidate projection')
 for shown,h in zip(result['hypotheses'],prepared['hypotheses']):
  start,end=h['startUtf8'],h['endUtf8']
  if not 0<=start<end<=len(raw) or raw[start:end].decode()!=h['sourceSpan']:raise ValueError('Comparison source span mismatch')
  shown['candidateComparisons']=[dict(originalSource=source,replacementSource=(raw[:start]+spelling.encode()+raw[end:]).decode()) for spelling in h['candidateSpellings']]
 return result
