#!/usr/bin/env bash
# detect.sh — resolve the project root by walking up from a start dir.
# Depends on core.sh registry arrays (APP_TYPES, APP_MARKERS_FN).

# resolve_root START_DIR
#   Walks up from START_DIR to /. At the first dir where a registered app's
#   markers fn returns 0, sets globals PROJECT_ROOT + APP_TYPE and returns 0.
#   Returns 1 if no registered project root is found anywhere up-tree.
PROJECT_ROOT=""
APP_TYPE=""

resolve_root() {
  local start dir i markers_fn
  start="$1"
  # normalize to an absolute existing dir
  dir="$(cd "$start" 2>/dev/null && pwd)" || return 1

  while :; do
    for i in "${!APP_TYPES[@]}"; do
      markers_fn="${APP_MARKERS_FN[$i]}"
      if "$markers_fn" "$dir"; then
        PROJECT_ROOT="$dir"
        APP_TYPE="${APP_TYPES[$i]}"
        return 0
      fi
    done
    [ "$dir" = "/" ] && break
    dir="$(dirname "$dir")"
  done
  return 1
}
