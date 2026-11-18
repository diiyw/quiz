#!/usr/bin/env python3
"""Run desktop Go commands with Quiz's pinned engine startup fix."""
import subprocess
import sys

from engine_patch import ROOT, patched_engine


def main():
    arguments = sys.argv[1:]
    check_font = "--check-ui-font" in arguments
    if check_font:
        arguments.remove("--check-ui-font")
    if not arguments or arguments[0] not in ("build", "run", "test", "vet"):
        raise SystemExit("usage: go.py [--check-ui-font] build|run|test|vet [Go arguments]")
    with patched_engine(check_font=check_font) as modfile:
        result = subprocess.run(["go", arguments[0], "-modfile", str(modfile), *arguments[1:]], cwd=ROOT)
    raise SystemExit(result.returncode)


if __name__ == "__main__":
    main()
