#!/usr/bin/env python3
"""Exercise binaries from the actual release package on a native runner."""
import argparse
import os
from pathlib import Path
import plistlib
import subprocess
import sys
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parent.parent


def smoke(binary):
    subprocess.run([sys.executable, str(ROOT / "scripts/smoke.py"), str(binary)], check=True)
    notices = subprocess.check_output([str(binary), "--licenses"], text=True, encoding="utf-8")
    for dependency in ["github.com/onix-engine/onix/LICENSE", "github.com/mholt/archives/LICENSE",
                       "github.com/bodgit/sevenzip/LICENSE", "github.com/nwaples/rardecode/v2/LICENSE",
                       "assets/QuizUI-OFL.txt"]:
        assert dependency in notices, f"Missing embedded license: {dependency}"


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
    with tempfile.TemporaryDirectory(prefix="quiz-package-") as temporary:
        directory = Path(temporary)
        if args.os == "linux":
            with tarfile.open(package) as archive:
                archive.extractall(directory, filter="data")
            installed = next(directory.iterdir())
            assert {item.name for item in directory.iterdir()} == {installed.name}
            assert {item.name for item in installed.iterdir()} == {"quiz"}
            smoke(installed / "quiz")
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
                assert {item.name for item in mount.iterdir() if not item.name.startswith(".")} == {"Quiz.app", "Applications"}
                subprocess.run(["codesign", "--verify", "--deep", "--strict",
                                str(mount / "Quiz.app")], check=True)
                smoke(contents / "MacOS/quiz")
            finally:
                subprocess.run(["hdiutil", "detach", str(mount)], check=True)
        else:
            # A fresh native runner lets us verify real per-user installation,
            # Start Menu registration, binaries and uninstall cleanup.
            installed = directory / "installed"
            log = directory / "install.log"
            uninstall_log = directory / "uninstall.log"
            try:
                msiexec("/i", str(package), "/qn", "/norestart", f"INSTALLFOLDER={installed}",
                        "/l*v", str(log))
                assert (Path(os.environ["APPDATA"]) / "Microsoft/Windows/Start Menu/Programs/Quiz/Quiz.lnk").is_file()
                assert {item.name for item in installed.iterdir()} == {"quiz.exe"}
                smoke(installed / "quiz.exe")
            except Exception:
                if log.exists():
                    print(log.read_text(encoding="utf-16", errors="replace"))
                raise
            finally:
                msiexec("/x", str(package), "/qn", "/norestart", "/l*v", str(uninstall_log))
            if (installed / "quiz.exe").exists():
                print(uninstall_log.read_text(encoding="utf-16", errors="replace"))
                raise AssertionError("Uninstall left quiz.exe in the installation directory")
            assert not (Path(os.environ["APPDATA"]) / "Microsoft/Windows/Start Menu/Programs/Quiz/Quiz.lnk").exists()
    print(f"Verified {package.name}")


if __name__ == "__main__":
    main()
