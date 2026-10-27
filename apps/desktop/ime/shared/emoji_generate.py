"""Generate the offline Chinese Emoji candidate index from pinned Unicode data."""

import hashlib
import json
import pathlib
import re
import sys
import xml.etree.ElementTree as ET


ROOT = pathlib.Path(__file__).resolve().parent
SOURCES = ROOT.parents[3] / "scripts/data/emoji"
MANIFEST = ROOT / "emoji_sources.json"
OUTPUT = ROOT / "emoji_data.h"
HAN_WORD = re.compile(r"[\u3400-\u9fff]{2,6}\Z")
MODIFIERS = set(range(0x1F3FB, 0x1F400))
# CLDR short names that also denote common, unrelated Pinyin actions.
REJECTED_TRIGGERS = {"注册", "一点", "笔记本"}


def digest(data):
    return hashlib.sha256(data).hexdigest()


def sequence_id(value):
    return "-".join(f"{ord(char):X}" for char in value)


def cpp_string(value):
    return json.dumps(value, ensure_ascii=False)


def main():
    manifest = json.loads(MANIFEST.read_text(encoding="utf-8"))
    if manifest["cldrVersion"] != "48.2" or manifest["emojiVersion"] != "15.0":
        raise ValueError("Unexpected Emoji source version")
    for filename, metadata in manifest["sources"].items():
        actual = digest((SOURCES / filename).read_bytes())
        if actual != metadata["sha256"]:
            raise ValueError(f"Pinned Emoji source digest mismatch: {filename}")

    canonical = {}
    for line in (SOURCES / "emoji-test-15.0.txt").read_text(encoding="utf-8").splitlines():
        match = re.match(r"^([0-9A-F ]+)\s*; fully-qualified\s+#", line)
        if match:
            value = "".join(chr(int(code, 16)) for code in match.group(1).split())
            canonical[value.replace("\ufe0f", "")] = value

    ranked = {}
    names = {}
    entries = []
    for filename in ("annotations-zh.xml", "annotations-derived-zh.xml"):
        annotations = ET.parse(SOURCES / filename).getroot()
        entries.extend(annotations.findall("./annotations/annotation"))
    for entry in entries:
        value = canonical.get(entry.attrib["cp"].replace("\ufe0f", ""))
        if not value or any(ord(char) in MODIFIERS for char in value):
            continue
        is_name = entry.attrib.get("type") == "tts"
        phrases = [entry.text.strip()] if is_name and entry.text else (entry.text or "").split("|")
        for raw in phrases:
            trigger = raw.strip()
            if is_name and trigger.startswith("旗: "):
                trigger = trigger.removeprefix("旗: ")
            if not HAN_WORD.fullmatch(trigger):
                continue
            if is_name:
                names.setdefault(value, set()).add(trigger)
            # A generic keyword must not propose a specific gender variant.
            if not is_name and ("男" in entry.text or "女" in entry.text):
                if ("男" in entry.text and "男" not in trigger) or ("女" in entry.text and "女" not in trigger):
                    continue
            rank = 0 if is_name else 1
            previous = ranked.setdefault(trigger, {}).get(value)
            ranked[trigger][value] = min(previous, rank) if previous is not None else rank

    alias_file = ROOT / manifest["aliases"]["file"]
    if digest(alias_file.read_bytes()) != manifest["aliases"]["sha256"]:
        raise ValueError("Reviewed Emoji alias digest mismatch")
    aliases = json.loads(alias_file.read_text(encoding="utf-8"))
    reviewed = {}
    for item in aliases:
        trigger = item["trigger"]
        value = item["emoji"]
        if trigger in reviewed or canonical.get(value.replace("\ufe0f", "")) != value:
            raise ValueError(f"Invalid reviewed Emoji mapping: {trigger}")
        reviewed[trigger] = value
    if len(reviewed) != manifest["aliases"]["count"]:
        raise ValueError("Reviewed Emoji alias count differs from the manifest")

    rows = []
    for trigger in sorted(set(ranked) | set(reviewed)):
        if trigger in REJECTED_TRIGGERS:
            continue
        if trigger in reviewed:
            values = [reviewed[trigger]]
        else:
            candidates = {
                value: rank for value, rank in ranked[trigger].items()
                if rank == 0 or any(
                    len(name) >= 2 and 0 < len(trigger) - len(name) <= 2 and
                    (trigger.startswith(name) or trigger.endswith(name))
                    for name in names.get(value, ())
                )
            }
            named = [value for value, rank in candidates.items() if rank == 0]
            if not named and len(candidates) > 2:
                continue
            if not candidates:
                continue
            values = sorted(named or candidates, key=sequence_id)[:1]
        rows.extend((trigger, value) for value in values)
    if len(rows) < 500:
        raise ValueError("Generated Emoji index is unexpectedly small")
    choices = dict(rows)
    if (choices.get("中国") != "🇨🇳" or choices.get("手机") != "📱" or
            any(word in choices for word in ("天气", "工作", "服务", "注册", "周末", "一点", "笔记本"))):
        raise ValueError("Emoji index failed ambiguity regressions: " + str({
            word: choices.get(word) for word in ("中国", "手机", "天气", "工作", "服务", "注册", "周末")
        }))

    header = "// Generated by emoji_generate.py from CLDR 48.2 and Unicode Emoji 15.0.\n"
    header += "// See emoji_sources.json and ranking_unicode_LICENSE.txt.\n"
    header += "static constexpr Mapping mappings[] = {\n"
    header += "".join(f"    {{{cpp_string(trigger)}, {cpp_string(value)}, {cpp_string(sequence_id(value))}}},\n" for trigger, value in rows)
    header += "};\n"
    if sys.argv[1:] == ["--check"]:
        if (len(rows) != manifest["generated"]["mappings"] or
                digest(header.encode("utf-8")) != manifest["generated"]["sha256"] or
                OUTPUT.read_text(encoding="utf-8") != header):
            raise ValueError("Generated Emoji index is stale")
    elif not sys.argv[1:]:
        OUTPUT.write_text(header, encoding="utf-8")
        manifest["generated"]["mappings"] = len(rows)
        manifest["generated"]["sha256"] = digest(header.encode("utf-8"))
        MANIFEST.write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    else:
        raise ValueError("Usage: python3 emoji_generate.py [--check]")
    print(f"Emoji index: {len(rows)} mappings across {len(set(key for key, _ in rows))} triggers")


if __name__ == "__main__":
    main()
