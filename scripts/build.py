#!/usr/bin/env python3
"""Build a native desktop binary, CLI and a ready-to-use distribution archive."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import plistlib
import re
import shutil
import subprocess
import tarfile
import zipfile

ROOT = Path(__file__).resolve().parent.parent


def run(*args, **kwargs):
    subprocess.run(args, cwd=ROOT, check=True, **kwargs)


def notices(destination):
    """Collect license/notice files from every module shipped by the Go graph."""
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
        target = destination / "licenses" / module["Path"].replace("/", "_")
        for item in source.iterdir():
            if item.is_file() and item.name.upper().startswith(("LICENSE", "COPYING", "NOTICE", "THIRD_PARTY")):
                target.mkdir(parents=True, exist_ok=True)
                shutil.copy2(item, target / item.name)
        if module["Path"] == "github.com/diiyw/onix":
            for relative in ["images/webp/LICENSE", "internal/brotli/LICENSE",
                             "font/resources/SKIA_LICENSE", "testdata/fonts/Noto_Sans_SC/OFL.txt"]:
                item = source / relative
                if item.is_file():
                    (target / relative).parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(item, target / relative)


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
        version = args.version.removeprefix("v").split("-")[0]
        if not re.fullmatch(r"\d+\.\d+\.\d+", version):
            version = "0.0.0"
        with (bundle / "Info.plist").open("wb") as f:
            plistlib.dump({"CFBundleIdentifier": "io.github.diiyw.quiz", "CFBundleName": "Quiz",
                          "CFBundleDisplayName": "Quiz", "CFBundleExecutable": "quiz",
                          "CFBundlePackageType": "APPL", "CFBundleShortVersionString": version,
                          "CFBundleVersion": version, "LSMinimumSystemVersion": "13.0",
                          "NSHighResolutionCapable": True}, f)
    if args.os == "windows":
        flags += " -H windowsgui -extldflags=-static"
    run("go", "build", "-trimpath", "-ldflags", flags, "-o", str(binary), ".", env=environment)
    environment["CGO_ENABLED"] = "0"
    cli_flags = f"-s -w -X github.com/diiyw/quiz/internal/cli.Version={args.version}"
    run("go", "build", "-trimpath", "-ldflags", cli_flags, "-o", str(stage / ("quiz-cli" + extension)), "./cmd/quiz-cli", env=environment)
    for filename in ["README.md", "THIRD_PARTY_NOTICES.md"]:
        shutil.copy2(ROOT / filename, stage / filename)
    notices(stage)
    if args.os == "darwin" and shutil.which("codesign"):
        run("codesign", "--force", "--deep", "--sign", "-", str(stage / "Quiz.app"))
        run("codesign", "--verify", "--deep", "--strict", str(stage / "Quiz.app"))
    if args.os == "windows":
        archive = dist / (name + ".zip")
        with zipfile.ZipFile(archive, "w", compression=zipfile.ZIP_DEFLATED) as z:
            for item in sorted(stage.rglob("*")):
                if item.is_file():
                    z.write(item, item.relative_to(stage.parent))
    else:
        archive = dist / (name + ".tar.gz")
        with tarfile.open(archive, "w:gz") as tar:
            tar.add(stage, arcname=name)
    with archive.open("rb") as stream:
        digest = hashlib.file_digest(stream, "sha256").hexdigest()
    archive.with_name(archive.name + ".sha256").write_text(f"{digest}  {archive.name}\n")
    print(archive)


if __name__ == "__main__":
    main()
