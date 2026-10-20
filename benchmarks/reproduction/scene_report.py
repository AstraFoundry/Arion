"""Shared scene provenance for diagnostic reports, independent of model inference."""
SCENE_LABELS = {
    'work': '工作',
    'prompt': '开发指令',
    'email': '邮件',
    'chat': '聊天',
    'list': '列表',
    'search': '搜索',
    'verbatim': '原样',
}

def scene_metadata(row, *, direct_scene_from_runner=False):
    assigned = row['scene']
    if 'effectiveModelScene' in row:
        effective = row['effectiveModelScene']
        evidence = 'recorded-in-inference-row'
    elif direct_scene_from_runner:
        effective = assigned
        evidence = 'reconstructed-from-frozen-runner-direct-scene-argument'
    else:
        raise ValueError('Missing effective model scene evidence')
    if assigned not in SCENE_LABELS or effective not in SCENE_LABELS:
        raise ValueError('Unknown scene')
    return dict(assignedScene=assigned, effectiveModelScene=effective,
                assignmentSource='fixture-metadata-not-automatic-detection',
                modelSceneEvidence=evidence, templateVersion=row['response']['template'])

def scene_markdown(metadata):
    assigned = metadata['assignedScene'];effective = metadata['effectiveModelScene']
    return [
        f"**Scene:** {SCENE_LABELS[assigned]} (`{assigned}`) → model: {SCENE_LABELS[effective]} (`{effective}`)",
        '',
        f"**Scene assignment:** `{metadata['assignmentSource']}`; model-scene evidence: `{metadata['modelSceneEvidence']}`.",
        '',
        f"**Prompt template:** `{metadata['templateVersion']}`.",
        '',
    ]
