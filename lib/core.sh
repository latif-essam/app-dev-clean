#!/usr/bin/env bash
# core.sh — output helpers, safe delete, app registry, interactive menu.
# Sourced by bin/devclean. bash 3.2 compatible (no associative arrays).

# --- pretty output ---
c_reset='\033[0m'; c_blue='\033[1;34m'; c_green='\033[1;32m'; c_yellow='\033[1;33m'; c_red='\033[1;31m'
step() { printf "${c_blue}==>${c_reset} %s\n" "$1"; }
ok()   { printf "${c_green}\xe2\x9c\x93${c_reset} %s\n" "$1"; }
warn() { printf "${c_yellow}!${c_reset} %s\n" "$1"; }
err()  { printf "${c_red}\xe2\x9c\x97${c_reset} %s\n" "$1" >&2; }

# nuke: rm -rf each path with logging; tolerates missing paths; never aborts.
nuke() {
  local p
  for p in "$@"; do
    if [ -e "$p" ]; then rm -rf "$p" && ok "removed $p"; else warn "skip (absent) $p"; fi
  done
}

# =================== GLOBAL targets (app-agnostic ~/ caches) ===================
# Live in core so they can run without a resolved project root.
clean_gradle_global() { step "GLOBAL Gradle cache (~/.gradle/caches)"; nuke "$HOME/.gradle/caches"; }
clean_xcode_dd()      { step "GLOBAL Xcode DerivedData"; nuke "$HOME/Library/Developer/Xcode/DerivedData"; }
clean_pods_cache()    { step "GLOBAL CocoaPods cache"; nuke "$HOME/Library/Caches/CocoaPods"; }

is_global_target() { case "$1" in gradle-global|xcode-dd|pods-cache) return 0 ;; *) return 1 ;; esac; }
run_global() {
  case "$1" in
    gradle-global) clean_gradle_global ;;
    xcode-dd)      clean_xcode_dd ;;
    pods-cache)    clean_pods_cache ;;
    *) warn "unknown global target: $1" ;;
  esac
}

# --- app registry (parallel arrays, indexed) ---
APP_TYPES=()        # e.g. "rn"
APP_MARKERS_FN=()   # fn(dir) -> 0 if dir is a root of this type
APP_MENU_FN=()      # fn() -> echoes menu rows (headers "#..." or "target|label|desc")
APP_RUN_FN=()       # fn(target) -> run one target
APP_POST_FN=()      # fn(target...) -> optional post-run hook (reinstall prompts); "" = none

# register_app TYPE MARKERS_FN MENU_FN RUN_FN [POST_FN]
register_app() {
  APP_TYPES+=("$1")
  APP_MARKERS_FN+=("$2")
  APP_MENU_FN+=("$3")
  APP_RUN_FN+=("$4")
  APP_POST_FN+=("${5:-}")
}

# app_index_for TYPE -> echoes index or nothing
app_index_for() {
  local i
  for i in "${!APP_TYPES[@]}"; do
    [ "${APP_TYPES[$i]}" = "$1" ] && { echo "$i"; return 0; }
  done
  return 1
}

# =================== interactive checklist (arrow keys) ===================
# Caller sets global MENU_ROWS (array of "#header" or "target|label|desc").
# Controls: up/down or j/k move, SPACE toggle, a all, n none, ENTER run, q quit.
# Writes chosen targets to global array SELECTED_TARGETS.
MENU_ROWS=()
SELECTED_TARGETS=()

menu() {
  local targets_all=() labels=() descs=() checked=()
  local render_map=()
  local row t l d
  for row in "${MENU_ROWS[@]}"; do
    if [[ "$row" == \#* ]]; then
      render_map+=("-1")
    else
      IFS='|' read -r t l d <<<"$row"
      targets_all+=("$t"); labels+=("$l"); descs+=("$d"); checked+=("0")
      render_map+=("$((${#targets_all[@]} - 1))")
    fi
  done

  local cursor=0
  local n=${#targets_all[@]}
  [ "$n" -eq 0 ] && { warn "no targets available"; SELECTED_TARGETS=(); return 0; }

  tput civis 2>/dev/null || true
  trap 'tput cnorm 2>/dev/null || true' RETURN

  local draw i idx mark pointer co key rest x
  draw() {
    tput clear 2>/dev/null || printf '\033[2J\033[H'
    printf "${c_blue}  devclean${c_reset}\n"
    printf "  up/down move . SPACE toggle . a all . n none . ENTER run . q quit\n\n"
    i=0
    for row in "${MENU_ROWS[@]}"; do
      idx="${render_map[$i]}"; i=$((i+1))
      if [ "$idx" = "-1" ]; then
        printf "\n  ${c_yellow}%s${c_reset}\n" "${row#\#}"
        continue
      fi
      mark=" "; [ "${checked[$idx]}" = "1" ] && mark="x"
      pointer="  "; co="$c_reset"
      if [ "$idx" -eq "$cursor" ]; then pointer="${c_green}> ${c_reset}"; co="$c_green"; fi
      printf "  %b[%s] ${co}%-14s${c_reset} %s\n" "$pointer" "$mark" "${labels[$idx]}" "${descs[$idx]}"
    done
  }

  while true; do
    draw
    IFS= read -rsn1 key
    if [ "$key" = $'\x1b' ]; then
      read -rsn2 rest
      key+="$rest"
    fi
    case "$key" in
      $'\x1b[A'|k) cursor=$(( (cursor - 1 + n) % n )) ;;
      $'\x1b[B'|j) cursor=$(( (cursor + 1) % n )) ;;
      ' ')        checked[$cursor]=$([ "${checked[$cursor]}" = "1" ] && echo 0 || echo 1) ;;
      a|A)        for x in "${!checked[@]}"; do checked[$x]=1; done ;;
      n|N)        for x in "${!checked[@]}"; do checked[$x]=0; done ;;
      q|Q)        tput cnorm 2>/dev/null || true; echo "bye"; exit 0 ;;
      ''|$'\n')   break ;;
    esac
  done
  tput cnorm 2>/dev/null || true

  SELECTED_TARGETS=()
  for x in "${!checked[@]}"; do
    [ "${checked[$x]}" = "1" ] && SELECTED_TARGETS+=("${targets_all[$x]}")
  done
}
