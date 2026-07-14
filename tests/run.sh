#!/usr/bin/env bash
# run.sh — dependency-free test harness for devclean core + detection logic.
# Usage: bash tests/run.sh
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HOME_DIR="$(dirname "$HERE")"
FIX="$HERE/fixtures"

# load library under test
source "$HOME_DIR/lib/core.sh"
source "$HOME_DIR/lib/detect.sh"
source "$HOME_DIR/apps/rn.sh"

pass=0; fail=0
check() { # check "desc" expected actual
  if [ "$2" = "$3" ]; then pass=$((pass+1)); printf "  ok   %s\n" "$1"
  else fail=$((fail+1)); printf "  FAIL %s (want '%s' got '%s')\n" "$1" "$2" "$3"; fi
}

# --- build fixtures fresh ---
rm -rf "$FIX"
mkdir -p "$FIX/rnapp/android" "$FIX/rnapp/ios" "$FIX/rnapp/src/deep/nested"
printf '{ "dependencies": { "react-native": "0.74.0" } }\n' > "$FIX/rnapp/package.json"
mkdir -p "$FIX/plainjs"
printf '{ "dependencies": { "express": "4.0.0" } }\n' > "$FIX/plainjs/package.json"
mkdir -p "$FIX/empty"

echo "== core =="
idx="$(app_index_for rn)"; check "app_index_for rn == 0" "0" "$idx"
check "app_index_for missing fails" "1" "$( app_index_for nope >/dev/null 2>&1; echo $? )"

tf="$FIX/tmpfile"; : > "$tf"
nuke "$tf" >/dev/null
check "nuke removes existing file (gone)" "1" "$( [ -e "$tf" ]; echo $? )"
check "nuke on absent path returns 0" "0" "$( nuke "$FIX/does-not-exist" >/dev/null; echo $? )"

echo "== rn markers =="
check "rn_markers rnapp" "0" "$( rn_markers "$FIX/rnapp"; echo $? )"
check "rn_markers plainjs (no react-native)" "1" "$( rn_markers "$FIX/plainjs"; echo $? )"
check "rn_markers empty" "1" "$( rn_markers "$FIX/empty"; echo $? )"

echo "== resolve_root =="
PROJECT_ROOT=""; APP_TYPE=""
resolve_root "$FIX/rnapp/src/deep/nested" >/dev/null 2>&1
check "resolve from nested subdir finds rnapp root" "$(cd "$FIX/rnapp" && pwd)" "$PROJECT_ROOT"
check "resolve sets APP_TYPE=rn" "rn" "$APP_TYPE"

PROJECT_ROOT=""; APP_TYPE=""
check "resolve in plainjs fails" "1" "$( resolve_root "$FIX/plainjs" >/dev/null 2>&1; echo $? )"
check "resolve in empty fails" "1" "$( resolve_root "$FIX/empty" >/dev/null 2>&1; echo $? )"

echo "== menu return code (set -e regression: menu must return 0) =="
MENU_ROWS=(); while IFS= read -r line; do MENU_ROWS+=("$line"); done < <(rn_menu_rows)
check "menu returns 0 on empty ENTER"      "0" "$( printf '\n'  | menu >/dev/null 2>&1; echo $? )"
check "menu returns 0 after SPACE+ENTER"   "0" "$( printf ' \n' | menu >/dev/null 2>&1; echo $? )"

rm -rf "$FIX"
echo
printf "PASS=%d FAIL=%d\n" "$pass" "$fail"
[ "$fail" -eq 0 ]
