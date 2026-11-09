#!/usr/bin/env python3
"""Subset the pinned Onix Noto Sans SC font for Quiz's static UI text.

Requires fonttools==4.55.8. Normal builds use the checked-in subset.
"""
import json
from pathlib import Path
import shutil
import subprocess

from fontTools import subset

ROOT = Path(__file__).resolve().parent.parent


def interface_characters():
    # Keep Latin, punctuation, and all non-ASCII characters in runtime messages.
    characters = set(range(0x20, 0x7F))
    characters.update(range(0x2000, 0x2070))
    characters.update(range(0x3000, 0x3040))
    characters.update(range(0xFF00, 0xFFF0))
    sources = [ROOT / "internal/desktop/view.html"]
    sources.extend(path for path in (ROOT / "internal").rglob("*.go")
                   if not path.name.endswith(("_test.go", "_generated.go")))
    for source in sources:
        characters.update(ord(character) for character in source.read_text(encoding="utf-8")
                          if ord(character) >= 0x80)
    return characters


def main():
    module = json.loads(subprocess.check_output(
        ["go", "list", "-m", "-json", "github.com/onix-engine/onix"], cwd=ROOT, text=True))
    source = Path(module["Dir"]) / "testdata/fonts/Noto_Sans_SC"
    options = subset.Options()
    options.recalc_timestamp = False
    options.layout_features = []
    options.name_IDs = [0, 1, 2, 3, 4, 5, 6, 13, 14]
    font = subset.load_font(str(source / "static/NotoSansSC-Regular.ttf"), options)
    subsetter = subset.Subsetter(options=options)
    subsetter.populate(unicodes=interface_characters())
    subsetter.subset(font)
    for record in font["name"].names:
        if record.nameID in (1, 3, 4, 6):
            record.string = ("QuizUI" if record.nameID == 6 else "Quiz UI").encode(record.getEncoding())
    output = ROOT / "assets/QuizUI.ttf"
    subset.save_font(font, str(output), options)
    shutil.copyfile(source / "OFL.txt", ROOT / "assets/QuizUI-OFL.txt")
    print(f"{output}: {output.stat().st_size:,} bytes")


if __name__ == "__main__":
    main()
