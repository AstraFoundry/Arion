"""Replay text-validator ablation without disabling runtime cancellation or identity checks."""
POLICIES=('strict','envelope','none')

def delivery(row,policy):
    if policy not in POLICIES:raise ValueError('Unknown delivery policy')
    response=row['response']
    if response.get('applicationTimeout'):
        return dict(accepted=False,origin='cleanup',outcome='timeout',text=response['cleanup']['text'])
    if response.get('cancelled'):
        return dict(accepted=False,origin='suppressed',outcome='cancelled',text=None)
    accepted=policy=='none' or (row['strict']=='accepted' if policy=='strict' else row['envelope']=='accepted')
    if accepted:return dict(accepted=True,origin='proposal',outcome='accepted-without-text-validation' if policy=='none' else 'accepted',text=response.get('text',''))
    return dict(accepted=False,origin='cleanup',outcome=row['strict'] if policy=='strict' else row['envelope'],text=response['cleanup']['text'])
