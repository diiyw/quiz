#!/usr/bin/env python3
"""Render the original SVG with resvg, then create native ICO and ICNS files.

Requires Node/npm and Pillow. Generated assets are committed, so release builds
do not need these tools. resvg preserves the SVG's blur and filter effects.
"""
from pathlib import Path
import subprocess
import tempfile

from PIL import Image

ROOT = Path(__file__).resolve().parent.parent


def main():
    with tempfile.TemporaryDirectory(prefix="quiz-icons-") as temporary:
        tools = Path(temporary)
        subprocess.run(["npm", "install", "--prefix", str(tools), "--no-audit",
                        "--no-fund", "@resvg/resvg-js@2.6.2"], check=True)
        png = subprocess.check_output([
            "node", "-e",
            "const fs = require('node:fs');"
            "const { Resvg } = require(process.argv[1]);"
            "process.stdout.write(new Resvg(fs.readFileSync(process.argv[2])).render().asPng());",
            str(tools / "node_modules" / "@resvg" / "resvg-js"),
            str(ROOT / "assets" / "icon.svg"),
        ])
    (ROOT / "assets" / "icon.png").write_bytes(png)
    with Image.open(ROOT / "assets" / "icon.png") as icon:
        icon.save(ROOT / "assets" / "icon.ico", sizes=[(n, n) for n in (16, 24, 32, 48, 64, 128, 256)])
        icon.save(ROOT / "assets" / "icon.icns")


if __name__ == "__main__":
    main()
