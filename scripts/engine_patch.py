"""Targeted startup fix for the pinned Onix release, without editing Go's cache.

Text2D constructors and UI themes eagerly load the system default even when
the HTML surface supplies a font. Use Quiz's embedded font for these defaults
as well. Glyph fallback is independent and remains enabled. Remove this patch
when Onix offers an application-wide default font configuration API.
"""
from contextlib import contextmanager
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parent.parent


def replace_once(source, old, new):
    if source.count(old) != 1:
        raise RuntimeError("Onix startup patch no longer matches; review engine_patch.py")
    return source.replace(old, new, 1)


@contextmanager
def patched_engine(check_font=False):
    module = json.loads(subprocess.check_output(
        ["go", "list", "-m", "-json", "github.com/onix-engine/onix"], cwd=ROOT, text=True))
    engine = Path(module["Dir"])
    replacements = {}
    text_path = engine / "text/font.go"
    text = replace_once(text_path.read_text(encoding="utf-8"),
                        '"github.com/onix-engine/onix/font"',
                        '"github.com/onix-engine/onix/font"\n\tquizassets "github.com/diiyw/quiz/assets"')
    replacements[text_path] = replace_once(text, "face, err := font.LoadDefault()",
                                           "face, err := font.Parse(quizassets.UIFont)")
    if check_font:
        # Desktop tests must use the embedded face, even on a machine with
        # working system fonts. Glyph fallback is separate and stays enabled.
        font_path = engine / "font/system.go"
        replacements[font_path] = replace_once(font_path.read_text(encoding="utf-8"),
            "func LoadDefault() (*Face, error) {",
            'func LoadDefault() (*Face, error) {\n\tpanic("Quiz UI loaded the system default font")')
    with tempfile.TemporaryDirectory(prefix="quiz-engine-") as temporary:
        directory = Path(temporary)
        # Go disallows overlays inside GOMODCACHE. Use a temporary local module
        # instead. Hard-link unchanged files when possible; never write to them.
        def link_or_copy(source, destination):
            # Windows cannot unlink read-only hard links without changing the
            # shared file attributes, so use independent writable copies there.
            if os.name == "nt":
                return shutil.copyfile(source, destination)
            try:
                os.link(source, destination)
            except OSError:
                shutil.copyfile(source, destination)

        local_engine = directory / "onix"
        shutil.copytree(engine, local_engine, copy_function=link_or_copy)
        for original, content in replacements.items():
            patched = local_engine / original.relative_to(engine)
            patched.parent.chmod(0o755)
            patched.unlink()
            patched.write_text(content, encoding="utf-8")
        modfile = directory / "quiz.mod"
        modfile.write_text((ROOT / "go.mod").read_text(encoding="utf-8")
                          + "\nreplace github.com/onix-engine/onix => "
                          + json.dumps(local_engine.as_posix()) + "\n", encoding="utf-8")
        shutil.copyfile(ROOT / "go.sum", modfile.with_suffix(".sum"))
        yield modfile
