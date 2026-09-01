#!/bin/bash
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Idempotent verify-or-repair for an agent identity configured via
# run-bot.sh.
#
# Usage:
#   bash verify-bot.sh <bot-id> [--help]
#
# What it does (in order, leaves working pieces alone):
#   1. Reads saved metadata from ~/.bots/<bot-id>/app.json
#   2. TOKEN: re-mints if missing or older than 50 minutes; otherwise
#      leaves the existing token alone.
#   3. REFRESHER: leaves alive process alone; restarts if dead/missing.
#   4. HARNESS: leaves a running harness session alone; relaunches via
#      the adapter if missing.
#
# Contrast with run-bot.sh: that script does clean-restart (kills +
# recreates everything). This script is verify-or-repair (leaves working
# pieces alone, only fixes broken ones). Safe to run repeatedly.
#
# Errors out if ~/.bots/<bot-id>/app.json doesn't exist (run run-bot.sh
# first to do the initial setup).

set -euo pipefail

case "${1:-}" in
  -h|--help)
    cat <<'HELP'
Idempotent verify-or-repair for an agent identity configured via
run-bot.sh.

Usage:
  bash verify-bot.sh <bot-id> [--help]

What it does (in order, leaves working pieces alone):
  1. Reads saved metadata from ~/.bots/<bot-id>/app.json
  2. TOKEN: re-mints if missing or older than 50 minutes; otherwise
     leaves the existing token alone.
  3. REFRESHER: leaves alive process alone; restarts if dead/missing.
  4. HARNESS: leaves a running harness session alone; relaunches via
     the adapter if missing.

Contrast with run-bot.sh: that script does clean-restart (kills +
recreates everything). This script is verify-or-repair (leaves working
pieces alone, only fixes broken ones). Safe to run repeatedly.

Errors out if ~/.bots/<bot-id>/app.json doesn't exist (run run-bot.sh
first to do the initial setup).
HELP
    exit 0
    ;;
esac

if [[ $# -ne 1 ]]; then
  cat >&2 <<USAGE
Usage: $(basename "$0") <bot-id>

Verifies (and repairs only as needed) the token, refresher, and harness
for an agent identity configured via run-bot.sh. Idempotent.

Reads saved metadata from ~/.bots/<bot-id>/app.json — run run-bot.sh
first to do the initial setup.
USAGE
  exit 2
fi

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=_lib/mint-token.sh
source "$SCRIPT_DIR/_lib/mint-token.sh"

GREEN=$'\033[0;32m'
YELLOW=$'\033[1;33m'
RED=$'\033[0;31m'
BLUE=$'\033[0;34m'
BOLD=$'\033[1m'
DIM=$'\033[2m'
RESET=$'\033[0m'

step() { echo -e "\n${BOLD}${BLUE}━━━ $1 ━━━${RESET}\n"; }
ok()    { echo -e "${GREEN}✓${RESET} $1"; }
warn()  { echo -e "${YELLOW}!${RESET} $1"; }
err()   { echo -e "${RED}✗${RESET} $1" >&2; }
note()  { echo -e "${DIM}$1${RESET}"; }

BOT_ID="$1"
if [[ ! "$BOT_ID" =~ ^[A-Za-z][A-Za-z0-9_-]{0,30}$ ]]; then
  err "<bot-id> must be 1-31 chars, start with a letter, alphanumeric + - / _"
  exit 1
fi

# ── Read saved metadata ──────────────────────────────────────────────
BOT_DIR="$HOME/.bots/$BOT_ID"
META_FILE="$BOT_DIR/app.json"
TOKEN_FILE="$BOT_DIR/token"
REFRESHER_LOG="$BOT_DIR/refresher.log"
REFRESHER_PID_FILE="$BOT_DIR/refresher.pid"

if [[ ! -f "$META_FILE" ]]; then
  err "No saved metadata for bot '$BOT_ID' at $META_FILE"
  err "Run run-bot.sh first to do the initial setup."
  exit 1
fi

read_meta() {
  python3 -c "import json,sys; d=json.load(open('$META_FILE')); print(d.get('$1', ''))"
}
APP_ID=$(read_meta app_id)
PEM_PATH=$(read_meta pem_path)
REPO_ROOT=$(read_meta repo_root)
APP_NAME=$(read_meta app_name)
ADAPTER_FROM_META=$(read_meta adapter)

# Adapter resolution: prefer env override, then metadata, then default.
ADAPTER="${ADAPTER:-${ADAPTER_FROM_META:-$SCRIPT_DIR/adapters/claude-code.sh}}"
if [[ ! -f "$ADAPTER" ]]; then
  err "Adapter not found: $ADAPTER"
  err "Set ADAPTER=<path> or re-run run-bot.sh to refresh metadata."
  exit 1
fi
# shellcheck source=adapters/claude-code.sh
source "$ADAPTER"

if [[ -z "$APP_ID" || -z "$PEM_PATH" || -z "$REPO_ROOT" ]]; then
  err "Saved metadata is incomplete: $META_FILE"
  err "Required: app_id, pem_path, repo_root. Re-run run-bot.sh to refresh."
  exit 1
fi
if [[ ! -f "$PEM_PATH" ]]; then
  err "Saved pem_path is no longer a file: $PEM_PATH"
  err "The .pem may have been moved/deleted. Re-run run-bot.sh."
  exit 1
fi
if [[ ! -d "$REPO_ROOT" ]]; then
  err "Saved repo_root is no longer a directory: $REPO_ROOT"
  err "The repo may have been moved. Re-run run-bot.sh."
  exit 1
fi

step "Verifying agent: $BOT_ID"
note "  app_id    = $APP_ID"
note "  app_name  = $APP_NAME"
note "  pem       = $PEM_PATH"
note "  repo_root = $REPO_ROOT"
note "  adapter   = $(basename "$ADAPTER")"

# ── Step 1: Token freshness ──────────────────────────────────────────
step "Token"

TOKEN=""
TOKEN_NEEDS_REFRESH=true

if [[ -f "$TOKEN_FILE" && -s "$TOKEN_FILE" ]]; then
  TOKEN_AGE=$(( $(date +%s) - $(stat -c %Y "$TOKEN_FILE") ))
  if (( TOKEN_AGE <= 3000 )); then  # 50 min
    TOKEN=$(<"$TOKEN_FILE")
    TOKEN_NEEDS_REFRESH=false
    ok "Token fresh (${TOKEN_AGE}s old, < 50min) — leaving alone"
  else
    warn "Token stale (${TOKEN_AGE}s old, > 50min) — will re-mint"
  fi
else
  warn "Token file missing or empty — will mint fresh"
fi

if $TOKEN_NEEDS_REFRESH; then
  if ! TOKEN=$(mint_installation_token "$PEM_PATH" "$APP_ID"); then
    err "Token mint failed."
    err "Most common cause: App not installed on the repo."
    err "Visit https://github.com/settings/apps/${APP_NAME} to verify."
    exit 1
  fi
  atomic_write "$TOKEN_FILE" "$TOKEN"
  ok "Token minted + saved: $TOKEN_FILE"
fi

# ── Step 2: Refresher liveness ───────────────────────────────────────
step "Refresher"

REFRESHER_NEEDS_RESTART=true

if [[ -f "$REFRESHER_PID_FILE" ]]; then
  PID=$(<"$REFRESHER_PID_FILE")
  if [[ -n "$PID" ]] && kill -0 "$PID" 2>/dev/null; then
    REFRESHER_NEEDS_RESTART=false
    ok "Refresher PID $PID alive — leaving alone"
    if [[ -f "$REFRESHER_LOG" ]]; then
      LAST_LOG_LINE=$(tail -1 "$REFRESHER_LOG" 2>/dev/null || echo "")
      [[ -n "$LAST_LOG_LINE" ]] && note "  last log: $LAST_LOG_LINE"
    fi
  else
    warn "Refresher PID file present but process $PID not alive — will restart"
  fi
else
  warn "No refresher PID file — will start fresh"
fi

if $REFRESHER_NEEDS_RESTART; then
  pkill -f "run-bot-refresher-${BOT_ID}" 2>/dev/null || true
  : > "$REFRESHER_LOG"

  nohup bash -c "
    set -uo pipefail
    cd /
    source '$SCRIPT_DIR/_lib/mint-token.sh'
    pem='$PEM_PATH'; app='$APP_ID'; bot_id='$BOT_ID'
    token_file='$TOKEN_FILE'; log='$REFRESHER_LOG'
    echo \"\$(date -u +%FT%TZ) refresher: starting (run-bot-refresher-\${bot_id}) [verify-bot.sh restart]\" >> \"\$log\"
    while sleep 2400; do
      if ! tmux has-session -t \"\$bot_id\" 2>/dev/null; then
        echo \"\$(date -u +%FT%TZ) refresher: harness session gone, exiting\" >> \"\$log\"
        exit 0
      fi
      if new_t=\$(mint_installation_token \"\$pem\" \"\$app\" 2>>\"\$log\"); then
        atomic_write \"\$token_file\" \"\$new_t\" 2>>\"\$log\" && \\
          echo \"\$(date -u +%FT%TZ) refresher: token rotated\" >> \"\$log\"
      else
        echo \"\$(date -u +%FT%TZ) refresher: mint failed, will retry next tick\" >> \"\$log\"
      fi
      if [[ -f \"\$log\" ]] && (( \$(stat -c %s \"\$log\" 2>/dev/null || echo 0) > 1048576 )); then
        tail -200 \"\$log\" > \"\$log.tmp\" && mv \"\$log.tmp\" \"\$log\"
      fi
    done
  " >>"$REFRESHER_LOG" 2>&1 &
  REFRESHER_PID=$!
  echo "$REFRESHER_PID" > "$REFRESHER_PID_FILE"
  disown "$REFRESHER_PID" 2>/dev/null || true
  ok "Refresher restarted (PID $REFRESHER_PID, log: $REFRESHER_LOG)"
fi

# ── Step 3: Harness session ──────────────────────────────────────────
step "Harness session"

if adapter_is_running "$BOT_ID"; then
  ok "Harness session $BOT_ID running — leaving alone"
else
  warn "Harness session $BOT_ID missing — relaunching via adapter"

  # Confirm the adapter's gh wrapper is still in place. Don't auto-reinstall
  # (that's run-bot.sh's job); fail with a clear message instead.
  GH_TARGET="$BOT_DIR/bin/gh"
  if [[ ! -f "$GH_TARGET" ]] || ! grep -q "bot-gh-wrapper marker" "$GH_TARGET" 2>/dev/null; then
    err "$GH_TARGET is missing or not the bot-gh-wrapper."
    err "Re-run run-bot.sh to reinstall the wrapper before verify-bot.sh can relaunch."
    exit 1
  fi

  adapter_launch "$BOT_ID" "$REPO_ROOT" "$TOKEN_FILE"
  ok "Harness session $BOT_ID launched"
fi

# ── Done ─────────────────────────────────────────────────────────────
echo
cat <<DONE
${BOLD}${GREEN}━━━ Verify complete. ━━━${RESET}

  Bot:           $BOT_ID
  App:           $APP_NAME (id $APP_ID)
  Adapter:       $(basename "$ADAPTER")
  Token file:    $TOKEN_FILE
  Refresher PID: $(<"$REFRESHER_PID_FILE")  (log: $REFRESHER_LOG)

${BOLD}Re-run safely:${RESET}
  bash $(basename "$0") $BOT_ID
  (verifies + repairs only what's broken)

${BOLD}Full clean restart (kills everything + recreates):${RESET}
  bash $(dirname "$0" | sed "s|$HOME|~|")/run-bot.sh

DONE
