#!/usr/bin/env python3
"""Exercise binaries from the actual release package on a native runner."""
import argparse
import hashlib
import os
from pathlib import Path
import plistlib
import subprocess
import sys
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parent.parent


def smoke(directory, desktop="quiz", cli="quiz-cli"):
    for binary in [directory / desktop, directory / cli]:
        subprocess.run([sys.executable, str(ROOT / "scripts/smoke.py"), str(binary)], check=True)


def msiexec(*arguments):
    result = subprocess.run(["msiexec.exe", *arguments])
    if result.returncode not in (0, 3010):
        raise RuntimeError(f"msiexec failed with exit code {result.returncode}")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--os", choices=["linux", "darwin", "windows"], required=True)
    args = parser.parse_args()
    extension = {"linux": ".tar.gz", "darwin": ".dmg", "windows": ".msi"}[args.os]
    packages = list((ROOT / "dist").glob("*" + extension))
    if len(packages) != 1:
        parser.error(f"expected one {extension} package in dist, got {len(packages)}")
    package = packages[0]
    digest = hashlib.sha256()
    with package.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    assert package.with_name(package.name + ".sha256").read_text().split()[0] == digest.hexdigest()
    with tempfile.TemporaryDirectory(prefix="quiz-package-") as temporary:
        directory = Path(temporary)
        if args.os == "linux":
            with tarfile.open(package) as archive:
                archive.extractall(directory, filter="data")
            installed = next(directory.iterdir())
            assert (installed / "quiz.svg").is_file()
            smoke(installed)
        elif args.os == "darwin":
            mount = directory / "mount"
            subprocess.run(["hdiutil", "attach", "-readonly", "-nobrowse", "-mountpoint",
                            str(mount), str(package)], check=True)
            try:
                contents = mount / "Quiz.app/Contents"
                with (contents / "Info.plist").open("rb") as stream:
                    info = plistlib.load(stream)
                assert (contents / "Resources" / info["CFBundleIconFile"]).is_file()
                assert (mount / "Applications").is_symlink()
                subprocess.run(["codesign", "--verify", "--deep", "--strict",
                                str(mount / "Quiz.app")], check=True)
                smoke(mount, "Quiz.app/Contents/MacOS/quiz")
            finally:
                subprocess.run(["hdiutil", "detach", str(mount)], check=True)
        else:
            # A fresh native runner lets us verify real per-user installation,
            # Start Menu registration, binaries and uninstall cleanup.
            installed = directory / "installed"
            log = directory / "install.log"
            try:
                msiexec("/i", str(package), "/qn", "/norestart", f"INSTALLFOLDER={installed}",
                        "/l*v", str(log))
                assert (Path(os.environ["APPDATA"]) / "Microsoft/Windows/Start Menu/Programs/Quiz/Quiz.lnk").is_file()
                assert (installed / "THIRD_PARTY_NOTICES.md").is_file()
                assert (installed / "licenses").is_dir()
                smoke(installed, "quiz.exe", "quiz-cli.exe")
            except Exception:
                if log.exists():
                    print(log.read_text(encoding="utf-16", errors="replace"))
                raise
            finally:
                msiexec("/x", str(package), "/qn", "/norestart")
            assert not (installed / "quiz.exe").exists()
            assert not (Path(os.environ["APPDATA"]) / "Microsoft/Windows/Start Menu/Programs/Quiz/Quiz.lnk").exists()
    print(f"Verified {package.name}")


if __name__ == "__main__":
    main()
