# devclean

Global dev-cache cleaner for macOS. Run it from **anywhere inside a project** — it
walks up from your current directory to find the real project root, then cleans that
project's caches. If you're not inside a recognized project, it refuses local cleanup
so it can never delete the wrong files.

Currently ships a **React Native** module (ported from the app's `scripts/clean.sh`).
New app types are drop-in: add one file under `apps/`.

## Install (local Homebrew tap)

```bash
# one-time: create a local tap
brew tap-new latif/tools

# copy this formula into the tap
cp Formula/devclean.rb "$(brew --repository latif/tools)/Formula/devclean.rb"

# install from the local git repo's HEAD
brew install --HEAD latif/tools/devclean
```

Update after editing the tool:

```bash
cd ~/dev-tools/devclean && git commit -am "..."
brew reinstall --HEAD latif/tools/devclean
```

## Usage

```bash
devclean                # interactive menu (inside a known project)
devclean ios js         # run named targets, no prompt
devclean local-all      # all LOCAL targets for the detected app type
devclean nuclear        # local-all + global caches + reinstall (confirmed)
devclean gradle-global  # a global cache target (allowed anywhere; confirmed)
devclean --root         # print resolved project root + app type
devclean --help
```

### React Native targets

- **LOCAL** (project, fast to rebuild): `android` `ios` `js` `metro` `watchman`
- **GLOBAL** (shared across all projects, slow): `gradle-global` `xcode-dd` `pods-cache`
- **COMBOS**: `local-all` `nuclear`

Global targets always prompt for confirmation because they affect every project on the
machine.

## Layout

```
bin/devclean      entrypoint: parse args -> resolve root -> dispatch
lib/core.sh       output helpers, nuke(), global cache targets, registry, menu
lib/detect.sh     walk-up project-root resolution
apps/rn.sh        React Native module (markers + targets)
Formula/devclean.rb   Homebrew formula
tests/run.sh      dependency-free test harness
```

## Adding a new app type

Create `apps/<type>.sh` defining four functions and registering them:

```bash
<type>_markers()   { # $1 = candidate dir; return 0 if it's a root of this type
}
<type>_menu_rows() { # echo "#Header" lines and "target|label|description" lines
}
<type>_run()       { # $1 = target name; do the cleanup
}
<type>_post()      { # optional: $@ = targets just run; post-run prompts (or omit)
}
register_app <type> <type>_markers <type>_menu_rows <type>_run <type>_post
```

That's it — detection, the menu, and CLI dispatch pick it up automatically. Global cache
targets (`gradle-global`, `xcode-dd`, `pods-cache`) live in `lib/core.sh` and are shared
across all app types.

## Testing

```bash
bash tests/run.sh
```

Covers the registry, `nuke`, RN markers, and up-tree root resolution (including the
refuse-when-not-a-project case).
