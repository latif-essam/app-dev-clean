#!/usr/bin/env bash
# rn.sh — React Native app module for devclean.
# Ported from superapp-frontend scripts/clean.sh (behavior unchanged).
# Local targets run with cwd == PROJECT_ROOT (entrypoint cd's there first).

# --- markers: is DIR the root of an RN project? ---
rn_markers() {
  local dir="$1"
  [ -f "$dir/package.json" ] || return 1
  { [ -d "$dir/android" ] || [ -d "$dir/ios" ]; } || return 1
  grep -q '"react-native"' "$dir/package.json" 2>/dev/null || return 1
  return 0
}

# =================== LOCAL targets ===================

rn_clean_android() {
  step "Android: build artifacts + gradle cache"
  if [ -x android/gradlew ]; then
    ( cd android && ./gradlew clean >/dev/null 2>&1 ) && ok "gradlew clean" || warn "gradlew clean failed (continuing)"
  fi
  nuke android/build android/app/build android/.gradle android/.cxx android/app/.cxx
}

rn_clean_ios() {
  step "iOS: Pods + build + lockfile"
  nuke ios/build ios/Pods ios/Podfile.lock
}

rn_clean_js() {
  step "JS: node_modules + package-lock.json"
  nuke node_modules package-lock.json
}

rn_clean_metro() {
  step "Metro/Haste temp caches"
  if command -v lsof >/dev/null 2>&1; then
    lsof -ti:8081 2>/dev/null | xargs kill -9 2>/dev/null && ok "killed Metro on :8081" || true
  fi
  find "${TMPDIR:-/tmp}" -maxdepth 1 \
    \( -name 'metro-*' -o -name 'haste-map-*' -o -name 'metro-cache' \) \
    -exec rm -rf {} + 2>/dev/null || true
  ok "metro caches cleared"
}

rn_clean_watchman() {
  step "Watchman: reset stale watch (fixes 'Recrawled N times' + phantom module-resolve errors)"
  if command -v watchman >/dev/null 2>&1; then
    watchman watch-del "$PROJECT_ROOT" >/dev/null 2>&1 || true
    watchman watch-project "$PROJECT_ROOT" >/dev/null 2>&1 || true
    ok "watchman re-crawled cleanly"
  else
    warn "watchman not installed; skipping"
  fi
}

# GLOBAL targets are app-agnostic and live in core.sh
# (clean_gradle_global / clean_xcode_dd / clean_pods_cache).

# =================== reinstall helpers ===================

rn_reinstall_js() {
  step "npm install"
  npm install && ok "node_modules restored"
}

rn_reinstall_pods() {
  if command -v pod >/dev/null 2>&1; then
    step "pod install --repo-update"
    ( cd ios && pod install --repo-update ) && ok "pods installed"
  else
    warn "CocoaPods not found; skipping pod install"
  fi
}

# =================== menu rows ===================

rn_menu_rows() {
  cat <<'ROWS'
#LOCAL (project — safe, fast to rebuild)
android|android|android/build, app/build, .gradle, .cxx + gradlew clean
ios|ios|ios/build, Pods, Podfile.lock
js|js|node_modules + package-lock.json
metro|metro|kill Metro :8081 + clear Metro/Haste temp caches
watchman|watchman|reset stale watch (fixes Recrawled / phantom resolve errors)
#GLOBAL (shared across ALL projects — slow to rebuild)
gradle-global|gradle cache|~/.gradle/caches
xcode-dd|xcode dd|Xcode DerivedData
pods-cache|pods cache|~/Library/Caches/CocoaPods
#COMBOS
local-all|local-all|android + ios + js + metro + watchman
nuclear|nuclear|everything + reinstall (npm install, pod install)
ROWS
}

# =================== dispatch ===================

rn_run() {
  case "$1" in
    android)        rn_clean_android ;;
    ios)            rn_clean_ios ;;
    js)             rn_clean_js ;;
    metro)          rn_clean_metro ;;
    watchman)       rn_clean_watchman ;;
    gradle-global)  clean_gradle_global ;;
    xcode-dd)       clean_xcode_dd ;;
    pods-cache)     clean_pods_cache ;;
    local-all)      rn_clean_android; rn_clean_ios; rn_clean_js; rn_clean_metro; rn_clean_watchman ;;
    nuclear)
      rn_clean_android; rn_clean_ios; rn_clean_js; rn_clean_metro; rn_clean_watchman
      clean_gradle_global; clean_xcode_dd; clean_pods_cache
      rn_reinstall_js; rn_reinstall_pods ;;
    *) warn "unknown target: $1" ;;
  esac
}

# =================== post-run hook (interactive reinstall offers) ===================
# rn_post TARGET...  — offer reinstall if js/ios cleaned but not via nuclear.
rn_post() {
  local targets=("$@")
  if printf '%s\n' "${targets[@]}" | grep -qx js && ! printf '%s\n' "${targets[@]}" | grep -qx nuclear; then
    read -r -p "  Run 'npm install' now? [y/N] " ri
    [[ "$ri" =~ ^[Yy]$ ]] && rn_reinstall_js
  fi
  if printf '%s\n' "${targets[@]}" | grep -qx ios && ! printf '%s\n' "${targets[@]}" | grep -qx nuclear; then
    read -r -p "  Run 'pod install' now? [y/N] " rp
    [[ "$rp" =~ ^[Yy]$ ]] && rn_reinstall_pods
  fi
  return 0
}

register_app rn rn_markers rn_menu_rows rn_run rn_post
