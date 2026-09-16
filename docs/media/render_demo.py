#!/usr/bin/env python3
"""Capture a disposable local npm fixture and render the README walkthrough."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import textwrap

from PIL import Image, ImageDraw, ImageFont


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--font", type=Path, required=True, help="A monospace TTF/OTF/TTC font")
    parser.add_argument("--output", type=Path, default=Path(__file__).parent)
    args = parser.parse_args()
    binary = args.binary.resolve()
    npm = shutil.which("npm")
    if not npm:
        parser.error("npm is required for the local fixture")
    args.output.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, NO_COLOR="1", npm_config_audit="false", npm_config_fund="false")
    version = subprocess.check_output([str(binary), "--version"], text=True).strip()

    with tempfile.TemporaryDirectory(prefix="adc-readme-") as temp:
        root = Path(temp) / "demo-app"
        root.mkdir()
        env["npm_config_cache"] = str(Path(temp) / "npm-cache")
        # A local dependency exercises real npm without registry downloads.
        dependency = root / "demo-expo"
        dependency.mkdir()
        (dependency / "package.json").write_text(json.dumps({"name": "expo", "version": "0.0.0"}))
        (root / "package.json").write_text(json.dumps({
            "name": "adc-demo", "version": "1.0.0", "private": True,
            "dependencies": {"expo": "file:./demo-expo"},
        }))
        (root / "App.js").write_text("export default function App() { return null; }\n")
        subprocess.run([npm, "install", "--offline", "--ignore-scripts"], cwd=root,
                       env=env, text=True, capture_output=True, check=True)
        (root / "node_modules" / "demo-cache.bin").write_bytes(b"0" * 32768)
        lock = root / "package-lock.json"
        lock_hash = hashlib.sha256(lock.read_bytes()).hexdigest()
        app_hash = hashlib.sha256((root / "App.js").read_bytes()).hexdigest()
        scenes = []

        def capture(title, note, command):
            result = subprocess.run([str(binary), *command], cwd=root, env=env,
                                    stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            if result.returncode:
                raise RuntimeError(result.stdout)
            output = result.stdout
            # Replace the longest spelling first when macOS resolves /var.
            for path in sorted({str(root), str(root.resolve())}, key=len, reverse=True):
                output = output.replace(path, "~/demo-app")
            scenes.append({"title": title, "note": note,
                           "command": "app-dev-clean " + " ".join(command), "output": output})

        capture("01 / Find the project", "Start inside your app. Check the detected root.", ["--root"])
        capture("02 / Preview the reset", "Dry-run lists the deletion and install command.",
                ["js", "--reinstall", "--dry-run", "-y"])
        assert (root / "node_modules" / "demo-cache.bin").exists(), "Dry-run changed files"
        capture("03 / Clean + reinstall", "Real npm output, streamed by app-dev-clean.",
                ["js", "--reinstall", "-y"])
        assert hashlib.sha256(lock.read_bytes()).hexdigest() == lock_hash, "Lockfile changed"
        assert hashlib.sha256((root / "App.js").read_bytes()).hexdigest() == app_hash, "Source changed"
        assert (root / "node_modules" / "expo" / "package.json").exists(), "Reinstall failed"
        assert not (root / "node_modules" / "demo-cache.bin").exists(), "Cache fixture survived"
        scenes.append({"title": "04 / Back to your project", "note": "Verified by the demo capture script.",
                       "command": "", "output": "package-lock.json   unchanged (SHA-256 checked)\n"
                       "App.js              unchanged (SHA-256 checked)\n"
                       "node_modules/expo   reinstalled from a local fixture\n\n"
                       "Preview first. Select the target you actually need."})

    metadata = {"version": version, "fixture": "Local demo Expo dependency; no registry downloads",
                "normalization": "Temporary project path replaced with ~/demo-app; playback paced for reading",
                "scenes": scenes}
    (args.output / "demo-transcript.json").write_text(json.dumps(metadata, indent=2) + "\n")
    regular = ImageFont.truetype(str(args.font), 20)
    small = ImageFont.truetype(str(args.font), 16)
    title_font = ImageFont.truetype(str(args.font), 29)
    width, height = 1280, 720
    frames, durations = [], []

    def frame(scene, visible):
        im = Image.new("RGB", (width, height), "#0c1420")
        draw = ImageDraw.Draw(im)
        draw.text((48, 28), "app-dev-clean", font=small, fill="#77e5bf")
        draw.text((48, 69), scene["title"], font=title_font, fill="#f3f7fc")
        draw.text((48, 113), scene["note"], font=small, fill="#b7c6d8")
        draw.rounded_rectangle((40, 164, 1240, 642), radius=14, fill="#111e2e", outline="#314054", width=2)
        for i, color in enumerate(["#e98b86", "#ddc183", "#77c5a5"]):
            draw.ellipse((60 + 22 * i, 184, 70 + 22 * i, 194), fill=color)
        draw.text((152, 179), "~/demo-app", font=small, fill="#9dacc0")
        draw.line((41, 212, 1238, 212), fill="#314054")
        y = 235
        if scene["command"]:
            draw.text((62, y), "$ " + scene["command"], font=regular, fill="#77e5bf")
            y += 42
        for line in visible:
            color = "#f3f7fc" if line.startswith(("==>", "Done.", "Dry run")) else "#c0cede"
            draw.text((62, y), line, font=regular, fill=color)
            y += 28
        label = "REAL CLI OUTPUT" if scene["command"] else "FIXTURE VERIFICATION"
        draw.text((48, 674), label + "  /  LOCAL DEMO FIXTURE  /  CURRENT SOURCE", font=small, fill="#9dacc0")
        return im

    for index, scene in enumerate(scenes):
        lines = []
        for line in scene["output"].strip().splitlines():
            lines.extend(textwrap.wrap(line, width=94, replace_whitespace=False, drop_whitespace=False) or [""])
        if len(lines) > 12:
            raise RuntimeError("Demo output no longer fits; update layout instead of clipping it")
        frames.append(frame(scene, []))
        durations.append(700)
        for line_count in range(1, len(lines) + 1):
            frames.append(frame(scene, lines[:line_count]))
            durations.append(180)
        durations[-1] = 4600
        if index == 1:
            frames[-1].save(args.output / "preview.png", optimize=True)
        if index == 2:
            frames[-1].save(args.output / "reinstall.png", optimize=True)

    palette_frames = [im.quantize(colors=96, method=Image.Quantize.MEDIANCUT) for im in frames]
    palette_frames[0].save(args.output / "walkthrough.gif", save_all=True,
                           append_images=palette_frames[1:], duration=durations,
                           loop=0, optimize=True, disposal=2)
    print(f"Captured {version}; checked dry-run, reinstall, source and lockfile preservation.")
    print(f"Wrote GIF, stills and transcript to {args.output.resolve()}")


if __name__ == "__main__":
    main()
