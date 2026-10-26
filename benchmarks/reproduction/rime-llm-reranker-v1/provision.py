"""Provision an isolated pinned research environment. Never edits installed Rime."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile
import urllib.request

STUDY = Path(__file__).resolve().parent
parser = argparse.ArgumentParser()
parser.add_argument("--workspace", required=True, type=Path)
args = parser.parse_args()
if sys.version_info < (3, 12):
    raise RuntimeError("Provisioning requires Python 3.12+; set EVAL_PYTHON explicitly")
workspace = args.workspace.resolve()
workspace.mkdir(parents=True, exist_ok=False)
sources = workspace / "sources"
sources.mkdir()
log = (workspace / "provision.log").open("x")


def run(command):
    log.write(json.dumps(command) + "\n")
    log.flush()
    subprocess.run(command, check=True, stdout=log, stderr=log)


def clone(url, revision, destination):
    run(["git", "clone", url, str(destination)])
    run(["git", "-C", str(destination), "checkout", "--detach", revision])


try:
    archive = sources / "lua.tar.gz"
    urllib.request.urlretrieve("https://www.lua.org/ftp/lua-5.4.8.tar.gz", archive)
    assert hashlib.sha256(archive.read_bytes()).hexdigest() == "4f18ddae154e793e46eeab727c59ef1c0c0c2b744e7b94219710d76f530629ae"
    with tarfile.open(archive) as stream:
        stream.extractall(sources, filter="data")
    clone("https://github.com/lunarmodules/luasocket.git", "95b7efa9da506ef968c1347edf3fc56370f0deed", sources / "luasocket")
    clone("https://github.com/LuaDist/dkjson.git", "e72ba0c9f5d8b8746fc306f6189a819dbb5cd0be", sources / "dkjson")
    clone("https://github.com/jundot/omlx.git", "6aba0dd3f28c690230a572a0c96cb43ee4f3d738", workspace / "omlx")
    run(["make", "-C", str(sources / "lua-5.4.8"), "macosx", "-j4"])
    variables = ["LUAV=5.4", f"LUAINC_macosx={sources / 'lua-5.4.8/src'}", f"LUAPREFIX_macosx={workspace / 'lua'}"]
    run(["make", "-C", str(sources / "luasocket"), "macosx", *variables])
    run(["make", "-C", str(sources / "luasocket"), "install", "PLAT=macosx", *variables])
    shutil.copy2(sources / "dkjson/dkjson.lua", workspace / "lua/share/lua/5.4/dkjson.lua")
    run(["xcrun", "clang", "-O2", "-bundle", "-undefined", "dynamic_lookup", "-I", str(sources / "lua-5.4.8/src"), str(STUDY / "clock.c"), "-o", str(workspace / "lua/lib/lua/5.4/benchclock.so")])
    run(["uv", "venv", "--python", "3.12", str(workspace / "venv")])
    # Git-based MLX packages are pinned by the oMLX source. Constrain all plain
    # packages to the actual measured environment rather than resolving anew.
    run(["uv", "pip", "install", "--python", str(workspace / "venv/bin/python"), "--constraint", str(STUDY / "constraints.txt"), str(workspace / "omlx")])
    url = "https://huggingface.co/api/models/mlx-community/Qwen3.5-0.8B-bf16/revision/3067585164dbcc505eec73d554349de6a27571a4?blobs=true"
    urllib.request.urlretrieve(url, workspace / "model-blobs.json")
    run([sys.executable, str(STUDY / "acquire.py"), str(workspace)])
    run(["node", str(STUDY / "prepare-native.mjs"), str(workspace)])
    (workspace / "provision-status.json").write_text(json.dumps({"status": "completed"}) + "\n")
except Exception as error:
    (workspace / "provision-status.json").write_text(json.dumps({"status": "failed", "error": str(error)}) + "\n")
    raise
