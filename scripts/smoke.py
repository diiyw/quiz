#!/usr/bin/env python3
"""Exercise the built executable with an actual archive, without a display."""
import argparse
from pathlib import Path
import subprocess
import tempfile
import zipfile

parser = argparse.ArgumentParser()
parser.add_argument("binary", type=Path)
args = parser.parse_args()
binary = str(args.binary.resolve())
subprocess.run([binary, "--version"], check=True)
with tempfile.TemporaryDirectory(prefix="quiz smoke ") as directory:
    root = Path(directory)
    archive = root / "中文 archive.zip"
    with zipfile.ZipFile(archive, "w") as z:
        z.writestr("folder/hello.txt", "Quiz extraction verified.")
    for destination in ["中文 archive", "中文 archive (2)"]:
        subprocess.run([binary, "--headless", "--", str(archive)], check=True)
        assert (root / destination / "folder" / "hello.txt").read_text() == "Quiz extraction verified."
print("Binary extraction and collision handling passed.")
