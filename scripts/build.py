#!/usr/bin/env python3
"""Package only the desktop app as Linux tar.gz, macOS DMG or Windows MSI."""
import argparse
from contextlib import contextmanager, nullcontext
import json
import os
from pathlib import Path
import plistlib
import re
import shutil
import subprocess
import tarfile

from native_packages import numeric_version, package_dmg, package_msi, windows_resources

ROOT = Path(__file__).resolve().parent.parent


def run(*args, **kwargs):
    subprocess.run(args, cwd=ROOT, check=True, **kwargs)


def notices():
    """Collect license texts for embedding without shipping loose files."""
    sections = [(ROOT / "THIRD_PARTY_NOTICES.md").read_text(encoding="utf-8"),
                "\n\n===== assets/QuizUI-OFL.txt =====\n\n"
                + (ROOT / "assets/QuizUI-OFL.txt").read_text(encoding="utf-8")]
    data = subprocess.check_output(["go", "list", "-m", "-json", "all"], cwd=ROOT, text=True)
    decoder = json.JSONDecoder()
    while data.strip():
        data = data.lstrip()
        module, end = decoder.raw_decode(data)
        data = data[end:]
        if module.get("Main"):
            continue
        directory = module.get("Replace", module).get("Dir")
        if not directory:
            continue
        source = Path(directory)
        files = []
        for item in sorted(source.iterdir()):
            if item.is_file() and item.name.upper().startswith(("LICENSE", "COPYING", "NOTICE", "THIRD_PARTY")):
                files.append(item)
        if module["Path"] == "github.com/onix-engine/onix":
            for relative in ["images/webp/LICENSE", "internal/brotli/LICENSE",
                             "font/resources/SKIA_LICENSE"]:
                item = source / relative
                if item.is_file():
                    files.append(item)
        for item in files:
            sections.append(f"\n\n===== {module['Path']}/{item.relative_to(source).as_posix()} =====\n\n"
                            + item.read_text(encoding="utf-8"))
    return "".join(sections)


@contextmanager
def embedded_notices():
    """Include full dependency licenses in the binary and clean up build inputs."""
    source = ROOT / "internal/cli/third_party_notices_generated.go"
    if source.exists():
        raise RuntimeError(f"Refusing to overwrite {source}")
    content = json.dumps(notices(), ensure_ascii=False)
    try:
        source.write_text(f"package cli\n\nfunc init() {{ ThirdPartyNotices = {content} }}\n", encoding="utf-8")
        yield
    finally:
        source.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--version", default="dev")
    parser.add_argument("--os", choices=["linux", "windows", "darwin"], required=True)
    parser.add_argument("--arch", choices=["amd64", "arm64"], required=True)
    args = parser.parse_args()
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9.+-]*", args.version):
        parser.error("version must contain only letters, digits, dots, plus or hyphens")
    name = f"quiz-{args.version}-{args.os}-{args.arch}"
    stage = ROOT / "build" / name
    if stage.exists():
        shutil.rmtree(stage)
    stage.mkdir(parents=True)
    dist = ROOT / "dist"
    dist.mkdir(exist_ok=True)
    environment = os.environ.copy()
    environment.update(GOOS=args.os, GOARCH=args.arch, CGO_ENABLED="1")
    flags = f"-s -w -X github.com/diiyw/quiz/internal/cli.Version={args.version}"
    extension = ".exe" if args.os == "windows" else ""
    binary = stage / ("quiz" + extension)
    if args.os == "darwin":
        environment.setdefault("MACOSX_DEPLOYMENT_TARGET", "13.0")
        bundle = stage / "Quiz.app" / "Contents"
        (bundle / "MacOS").mkdir(parents=True)
        binary = bundle / "MacOS" / "quiz"
        version = numeric_version(args.version)
        (bundle / "Resources").mkdir()
        shutil.copy2(ROOT / "assets/icon.icns", bundle / "Resources/Quiz.icns")
        with (bundle / "Info.plist").open("wb") as f:
            plistlib.dump({"CFBundleIdentifier": "io.github.diiyw.quiz", "CFBundleName": "Quiz",
                          "CFBundleDisplayName": "Quiz", "CFBundleExecutable": "quiz",
                          "CFBundlePackageType": "APPL", "CFBundleShortVersionString": version,
                          "CFBundleIconFile": "Quiz.icns",
                          "CFBundleVersion": version, "LSMinimumSystemVersion": "13.0",
                          "NSHighResolutionCapable": True}, f)
    if args.os == "windows":
        flags += " -H windowsgui -extldflags=-static"
    with embedded_notices(), windows_resources(args.arch) if args.os == "windows" else nullcontext():
        run("go", "build", "-trimpath", "-ldflags", flags, "-o", str(binary), ".", env=environment)
    if args.os == "darwin" and shutil.which("codesign"):
        run("codesign", "--force", "--deep", "--sign", "-", str(stage / "Quiz.app"))
        run("codesign", "--verify", "--deep", "--strict", str(stage / "Quiz.app"))
    if args.os == "windows":
        archive = dist / (name + ".msi")
        package_msi(stage, archive, args.version, args.arch)
    elif args.os == "darwin":
        archive = dist / (name + ".dmg")
        package_dmg(stage, archive)
    else:
        archive = dist / (name + ".tar.gz")
        with tarfile.open(archive, "w:gz") as tar:
            tar.add(stage, arcname=name)
    print(archive)


if __name__ == "__main__":
    main()
