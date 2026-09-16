# README media

The cover is a conceptual illustration. The terminal walkthrough is captured
from the actual CLI and npm, then typeset into frames for readability. It is not
a screen recording or a performance comparison.

## Assets

| File | Purpose |
| --- | --- |
| `cover.png` | README cover; conceptual illustration, not product UI |
| `walkthrough.gif` | Four-step preview / cleanup / reinstall walkthrough |
| `preview.png` | Static alternative for the preview step |
| `reinstall.png` | Static view of cleanup and npm output |
| `demo-transcript.json` | Captured output, version, and normalization details |
| `render_demo.py` | Reproducible fixture capture and media renderer |

The capture uses a local package named `expo` only to exercise detection and npm's
locked install. It does not download or install Expo from the registry. The fixture
has a small synthetic cache file; file sizes and timings do not represent a real
project. Temporary paths are replaced with `~/demo-app`, and playback is paced for
reading. The final scene reports checks performed by the capture script, not CLI
output. Each capture gets a new temporary project and npm cache.

## Regenerate the walkthrough

Requires the compiled CLI, Node/npm, Python 3, Pillow, and a monospace font.
Pillow is a documentation-tool dependency only. From the repository root:

```sh
go build -o dist/dev/app-dev-clean .
python3 -m venv /tmp/adc-media-venv
/tmp/adc-media-venv/bin/python -m pip install Pillow
/tmp/adc-media-venv/bin/python docs/media/render_demo.py \
  --binary dist/dev/app-dev-clean \
  --font /System/Library/Fonts/Menlo.ttc
```

Use your local monospace font path on Linux; for example a DejaVu Sans Mono TTF.
The capture script uses macOS/Linux subprocess invocation; Windows CLI behavior
is covered separately by project tests and the planned installation checks.

The script checks that dry-run preserves the fixture, reinstall succeeds, and
the source/lockfile hashes do not change. It fails instead of clipping output
that no longer fits. Review the generated stills, GIF, and transcript before
committing them. Update the README's captured version when refreshing the assets.

## Cover prompt

The cover was generated with the built-in imagegen tool using this prompt:

> Create a polished wide 3:1 GitHub README cover for an open-source terminal tool named app-dev-clean. Use case: ads-marketing. This is a conceptual editorial illustration, NOT an application screenshot. Dark midnight navy canvas, off-white crisp large monospaced title exactly 'app-dev-clean', smaller subtitle exactly 'A clearer reset for mobile development'. Below it a tiny typographic row exactly 'PREVIEW  /  CLEAN  /  REINSTALL'. On the right a beautifully restrained isometric arrangement of terminal window outlines, stacked cache folders and one intact document with a small lock icon, showing scattered build-cache blocks becoming neatly arranged. Emerald and muted ice-blue accents, precise geometry, refined technical manual feeling, generous whitespace, sharp readable text. No invented terminal output, no numbers or performance claims, no third-party logos, no badges, no robot imagery, no gradients overwhelming text, no extra text. Export a high-resolution horizontal banner.
