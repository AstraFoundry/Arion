"""Exercise Darwin directory exchange on owned fixtures, never installed sources."""
import ctypes
import errno
import hashlib
import json
import multiprocessing
import os
from pathlib import Path
import platform
import subprocess
import time
import uuid

RENAME_SWAP = 0x00000002
RENAME_NOFOLLOW_ANY = 0x00000010

def observe(path, stop, output):
    counts = {"exists": 0, "missing": 0, "errors": []}
    while not stop.is_set():
        try:
            os.stat(path)
            counts["exists"] += 1
        except FileNotFoundError:
            counts["missing"] += 1
        except OSError as exc:
            counts["errors"].append(exc.errno)
    output.put(counts)

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def main():
    root = Path.cwd()
    scratch = root / ".workspace/notes/wechat-ime-crash/lifecycle-repair" / str(uuid.uuid4())
    scratch.mkdir(parents=True, mode=0o700)
    installed = scratch / "installed.app"
    candidate = scratch / "candidate.app"
    for bundle, content in [(installed, b"previous-complete-build"), (candidate, b"next-complete-build")]:
        (bundle / "Contents/MacOS").mkdir(parents=True)
        (bundle / "Contents/MacOS/fixture").write_bytes(content)
        (bundle / "Contents/Info.plist").write_bytes(b"owned non-executable fixture")
    old_hash = digest(installed / "Contents/MacOS/fixture")
    new_hash = digest(candidate / "Contents/MacOS/fixture")
    library = ctypes.CDLL("/usr/lib/libSystem.B.dylib", use_errno=True)
    rename = library.renameatx_np
    rename.argtypes = [ctypes.c_int, ctypes.c_char_p, ctypes.c_int, ctypes.c_char_p, ctypes.c_uint]
    rename.restype = ctypes.c_int
    fd = os.open(scratch, os.O_RDONLY | os.O_DIRECTORY)
    def exchange(destination=b"candidate.app", flags=RENAME_SWAP | RENAME_NOFOLLOW_ANY):
        ctypes.set_errno(0)
        result = rename(fd, b"installed.app", fd, destination, flags)
        return {"return": result, "errno": ctypes.get_errno()}
    before = {"installed": old_hash, "candidate": new_hash}
    first = exchange()
    after = {"installed": digest(installed / "Contents/MacOS/fixture"), "candidate": digest(candidate / "Contents/MacOS/fixture")}
    rollback = exchange()
    restored = {"installed": digest(installed / "Contents/MacOS/fixture"), "candidate": digest(candidate / "Contents/MacOS/fixture")}
    missing_target = exchange(b"absent.app")
    after_error = {"installed": digest(installed / "Contents/MacOS/fixture"), "candidate": digest(candidate / "Contents/MacOS/fixture")}
    context = multiprocessing.get_context("spawn")
    stop = context.Event()
    output = context.Queue()
    reader = context.Process(target=observe, args=(str(installed), stop, output))
    reader.start()
    time.sleep(0.15)
    failures = []
    for _ in range(2000):
        result = exchange()
        if result["return"] != 0:
            failures.append(result)
            break
    stop.set()
    reader.join(10)
    if reader.is_alive():
        reader.terminate()
        raise RuntimeError("Owned observer did not exit")
    observed = output.get(timeout=3)
    os.close(fd)
    report = {
        "scope": "Owned unsigned directory fixtures only; no source API, installed bundle, application cache, or running IME modified",
        "system": {"macOS": platform.mac_ver()[0], "build": subprocess.check_output(["sw_vers", "-buildVersion"], text=True).strip(), "machine": platform.machine()},
        "sameDeviceAsInstalledParent": scratch.stat().st_dev == Path.home().joinpath("Library/Input Methods").stat().st_dev,
        "before": before, "promotion": first, "afterPromotion": after, "rollback": rollback, "afterRollback": restored,
        "missingTarget": missing_target, "afterError": after_error,
        "repeatedExchanges": 2000, "exchangeFailures": failures, "concurrentDestinationStat": observed,
        "limits": ["Only local fixture volume tested; no unsupported/cross-volume target or signed app validation tested", "Stat observations are finite sampling, not proof of absence of all namespace races", "Atomic name exchange does not establish crash/power-loss recovery, stable resource reads across multiple lookups, or TIS cache coherence"]
    }
    report["passed"] = first["return"] == 0 and rollback["return"] == 0 and after == {"installed": new_hash, "candidate": old_hash} and restored == before and missing_target == {"return": -1, "errno": errno.ENOENT} and after_error == before and not failures and observed["exists"] > 0 and observed["missing"] == 0 and not observed["errors"]
    report["sourceSHA256"] = hashlib.sha256(Path(__file__).read_bytes()).hexdigest()
    (scratch / "result.json").write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps({"result": str(scratch / "result.json"), **report}, indent=2))
    if not report["passed"]:
        raise SystemExit(1)

if __name__ == "__main__":
    main()
