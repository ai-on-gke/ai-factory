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

# Interactive walkthrough: provision a per-agent GitHub App identity end to
# end, then launch the configured harness adapter.
#
# Usage:
#   bash run-bot.sh [--help]
#
# Walks the operator through:
#   Step 1. Pick a bot-id, target repo, and local repo-root
#   Step 2. Create the GitHub App in the browser (script opens the form)
#   Step 3. Download the App's private key (.pem)
#   Step 4. Install the App on the target repo
#   Step 5. Capture the App ID
#   Step 6. Wire everything up + launch the harness adapter
#
# Re-run semantics: safe to run again with the same answers — kills any
# prior harness session + refresher and starts fresh.
#
# Adapter selection: by default sources adapters/claude-code.sh from the
# same directory. Override with ADAPTER=/path/to/adapter.sh to plug in a
# different harness.

set -euo pipefail

case "${1:-}" in
  -h|--help)
    cat <<'HELP'
Interactive walkthrough: provision a per-agent GitHub App identity end
to end, then launch the configured harness adapter.

Usage:
  bash run-bot.sh [--help]

Walks the operator through:
  Step 1. Pick a bot-id, target repo, and local repo-root
  Step 2. Create the GitHub App in the browser (script opens the form)
  Step 3. Download the App's private key (.pem)
  Step 4. Install the App on the target repo
  Step 5. Capture the App ID
  Step 6. Wire everything up + launch the harness adapter

Re-run semantics: safe to run again with the same answers — kills any
prior harness session + refresher and starts fresh.

Adapter selection: by default sources adapters/claude-code.sh from the
same directory. Override with ADAPTER=/path/to/adapter.sh to plug in a
different harness.
HELP
    exit 0
    ;;
esac

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=_lib/mint-token.sh
source "$SCRIPT_DIR/_lib/mint-token.sh"
ADAPTER="${ADAPTER:-$SCRIPT_DIR/adapters/claude-code.sh}"
if [[ ! -f "$ADAPTER" ]]; then
  echo "run-bot: adapter not found: $ADAPTER" >&2
  exit 1
fi
# shellcheck source=adapters/claude-code.sh
source "$ADAPTER"

# ANSI-C quoting ($'...') so bash interprets \033 at definition time.
GREEN=$'\033[0;32m'
YELLOW=$'\033[1;33m'
RED=$'\033[0;31m'
BLUE=$'\033[0;34m'
BOLD=$'\033[1m'
DIM=$'\033[2m'
RESET=$'\033[0m'

step()  { echo -e "\n${BOLD}${BLUE}━━━ $1 ━━━${RESET}\n"; }
ok()    { echo -e "${GREEN}✓${RESET} $1"; }
warn()  { echo -e "${YELLOW}!${RESET} $1"; }
err()   { echo -e "${RED}✗${RESET} $1" >&2; }
note()  { echo -e "${DIM}$1${RESET}"; }
ask()   { echo -e "${BOLD}$1${RESET}"; }

# ── Welcome ──────────────────────────────────────────────────────────
clear || true
cat <<INTRO
${BOLD}Per-agent GitHub App identity setup${RESET}

This script will lead you through:

  1. Creating a GitHub App (the agent's identity)
  2. Downloading the App's private key
  3. Installing the App on a repo
  4. Wiring credentials into a refreshing token
  5. Launching the configured harness adapter

Adapter: ${BOLD}$(basename "$ADAPTER" .sh)${RESET} — $(adapter_describe)

You'll need a browser open alongside this terminal — most of the App
creation happens at github.com.

${DIM}Press Ctrl-C at any point to abort. Re-running this script is safe.${RESET}

INTRO

read -r -p "Press Enter to begin..." _

# ── Prereqs ──────────────────────────────────────────────────────────
step "Checking prerequisites"

missing=()
for cmd in curl openssl python3; do
  if command -v "$cmd" &>/dev/null; then
    ok "$cmd: $(command -v "$cmd")"
  else
    missing+=("$cmd")
  fi
done

if (( ${#missing[@]} > 0 )); then
  err "Missing core commands: ${missing[*]}"
  for m in "${missing[@]}"; do
    case "$m" in
      curl)    echo "  Install curl:    sudo apt install curl" ;;
      openssl) echo "  Install openssl: sudo apt install openssl" ;;
      python3) echo "  Install python3: sudo apt install python3" ;;
    esac
  done
  exit 1
fi

# Adapter prereqs (e.g. tmux, claude for the default adapter).
if ! adapter_check_prereqs; then
  exit 1
fi
ok "adapter prerequisites satisfied"

# ── Step 1: bot-id + repo ────────────────────────────────────────────
step "Step 1 — Pick an agent identity"

cat <<EXPLAIN
The ${BOLD}bot-id${RESET} is your local short name for this agent. It will be:
  • The harness session name (e.g. tmux session)
  • The folder name under ${DIM}~/.bots/${RESET}
  • A reference for re-running this script later

Pick something short and memorable, e.g. "speccer1", "reviewer-bot".
1-31 chars, must start with a letter, alphanumeric + - / _.

EXPLAIN
while true; do
  ask "bot-id:"
  read -r BOT_ID
  BOT_ID=$(echo "$BOT_ID" | xargs)
  if [[ "$BOT_ID" =~ ^[A-Za-z][A-Za-z0-9_-]{0,30}$ ]]; then
    break
  fi
  warn "Invalid. Must be 1-31 chars, start with a letter, alphanumeric + - / _."
done
ok "bot-id: $BOT_ID"

echo
cat <<EXPLAIN
The ${BOLD}repo${RESET} is the GitHub repository where this agent will work
(commit, open PRs, comment on issues). It must be a repo you have admin
access to (so you can install your new App there).

Format: ${DIM}owner/repo${RESET}, e.g. ${DIM}octocat/hello-world${RESET}.

EXPLAIN
while true; do
  ask "repo (owner/repo):"
  read -r REPO
  REPO=$(echo "$REPO" | xargs)
  if [[ "$REPO" =~ ^[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+$ ]]; then
    break
  fi
  warn "Invalid. Must be in owner/repo format."
done
ok "repo: $REPO"

echo
cat <<EXPLAIN
The ${BOLD}repo-root${RESET} is the local directory where the harness will run.
Typically a checkout of the repo above.

EXPLAIN
DEFAULT_REPO_ROOT=""
if [[ -d "$HOME/src/github.com/${REPO}" ]]; then
  DEFAULT_REPO_ROOT="$HOME/src/github.com/${REPO}"
fi
while true; do
  if [[ -n "$DEFAULT_REPO_ROOT" ]]; then
    ask "repo-root [default: $DEFAULT_REPO_ROOT]:"
  else
    ask "repo-root (absolute path):"
  fi
  read -r REPO_ROOT
  REPO_ROOT=$(echo "$REPO_ROOT" | xargs)
  REPO_ROOT="${REPO_ROOT:-$DEFAULT_REPO_ROOT}"
  REPO_ROOT="${REPO_ROOT/#\~/$HOME}"
  if [[ -d "$REPO_ROOT" ]]; then
    break
  fi
  warn "Not a directory: $REPO_ROOT"
done
ok "repo-root: $REPO_ROOT"

# ── Step 2: create the App ───────────────────────────────────────────
step "Step 2 — Create the GitHub App"

# Suffix includes minute precision so re-runs in the same day don't collide
# on App name.
APP_NAME="${BOT_ID}-bot-$(date +%y%m%d-%H%M)"

cat <<EXPLAIN
Now create the App in your browser. I'll open the "New GitHub App" form
with these fields pre-filled:

  Name:          ${BOLD}${APP_NAME}${RESET}
  Homepage URL:  https://github.com/${REPO}
  Public:        unchecked (private to your account)
  Webhook:       unchecked (not active)

${BOLD}You'll need to fill in${RESET} (script can't pre-fill these):

  Repository permissions — set to what your agent will do. A reasonable
  default for a coding agent:
    • Contents:       Read and write
    • Issues:         Read and write
    • Pull requests:  Read and write
    • Metadata:       Read-only (default)
    • Checks:         Read-only

  ${DIM}If unsure, this is fine — you can edit permissions later from the${RESET}
  ${DIM}App's settings page.${RESET}

  Where can this GitHub App be installed?
    Choose "Only on this account"

Then click ${BOLD}"Create GitHub App"${RESET} at the bottom.

EXPLAIN
read -r -p "Press Enter to open the browser..." _

URL="https://github.com/settings/apps/new"
URL+="?name=${APP_NAME}"
URL+="&url=https%3A%2F%2Fgithub.com%2F${REPO//\//%2F}"
URL+="&public=false"
URL+="&webhook_active=false"

if command -v xdg-open &>/dev/null; then
  xdg-open "$URL" >/dev/null 2>&1 &
elif command -v open &>/dev/null; then
  open "$URL"
else
  warn "Could not auto-open browser."
  echo "Visit this URL manually:"
  echo "  $URL"
fi

echo
note "If the browser didn't open, the URL is: $URL"
echo
read -r -p "Press Enter once you've clicked 'Create GitHub App' and you're on the App's settings page..." _

# ── Step 3: download the .pem ────────────────────────────────────────
step "Step 3 — Download the App's private key"

cat <<EXPLAIN
You should now be on your new App's settings page (URL looks like
${DIM}https://github.com/settings/apps/${APP_NAME}${RESET}).

${BOLD}On that page, do this:${RESET}

  1. Scroll down to the section "${BOLD}Private keys${RESET}"
  2. Click "${BOLD}Generate a private key${RESET}"
  3. A .pem file will download (typically to ~/Downloads/)

The .pem is a private RSA key. Keep it safe — anyone with this file
can act as your bot.

EXPLAIN
read -r -p "Press Enter once the .pem has downloaded..." _

# Best-effort default: newest matching .pem in Downloads.
DEFAULT_PEM=$(ls -t "${HOME}/Downloads/${APP_NAME}".*.private-key.pem 2>/dev/null | head -1 || true)
[[ -z "$DEFAULT_PEM" ]] && DEFAULT_PEM=$(ls -t "${HOME}/Downloads"/*.private-key.pem 2>/dev/null | head -1 || true)

while true; do
  if [[ -n "$DEFAULT_PEM" ]]; then
    ask "Path to the .pem [default: $DEFAULT_PEM]:"
  else
    ask "Path to the .pem (absolute path):"
  fi
  read -r PEM_PATH
  PEM_PATH=$(echo "$PEM_PATH" | xargs)
  PEM_PATH="${PEM_PATH:-$DEFAULT_PEM}"
  PEM_PATH="${PEM_PATH/#\~/$HOME}"
  if [[ ! -f "$PEM_PATH" ]]; then
    warn "Not a file: $PEM_PATH"
    continue
  fi
  if openssl rsa -in "$PEM_PATH" -check -noout 2>/dev/null; then
    break
  fi
  warn "File is not a valid RSA private key: $PEM_PATH"
done
ok ".pem: $PEM_PATH"

# ── Step 4: install the App on the repo ──────────────────────────────
step "Step 4 — Install the App on your repo"

cat <<EXPLAIN
A GitHub App must be ${BOLD}installed${RESET} on a repo before it can act there.

${BOLD}On the App's settings page${RESET} (where you are now):

  1. In the left sidebar, click "${BOLD}Install App${RESET}"
  2. Click "${BOLD}Install${RESET}" next to your account
  3. On the next screen, choose "${BOLD}Only select repositories${RESET}"
  4. Select: ${BOLD}${REPO}${RESET}
  5. Click "${BOLD}Install${RESET}"

EXPLAIN
read -r -p "Press Enter once the App is installed on $REPO..." _

# ── Step 5: capture the App ID ───────────────────────────────────────
step "Step 5 — Get the App ID"

cat <<EXPLAIN
The ${BOLD}App ID${RESET} is a numeric identifier GitHub assigned to your App.

${BOLD}Where to find it:${RESET}

  1. Go back to your App's settings page:
     ${DIM}https://github.com/settings/apps/${APP_NAME}${RESET}

  2. At the top of the page, near the App's name, you'll see:
     ${DIM}"App ID: 1234567"${RESET}

  3. Copy that number.

EXPLAIN
while true; do
  ask "App ID (numeric):"
  read -r APP_ID
  APP_ID=$(echo "$APP_ID" | xargs)
  if [[ "$APP_ID" =~ ^[0-9]+$ ]]; then
    break
  fi
  warn "App ID must be numeric (digits only)."
done
ok "App ID: $APP_ID"

# ── Step 6: wire everything up ───────────────────────────────────────
step "Step 6 — Wiring up the agent"

BOT_DIR="$HOME/.bots/$BOT_ID"
mkdir -p "$BOT_DIR"
chmod 700 "$HOME/.bots" "$BOT_DIR"

TOKEN_FILE="$BOT_DIR/token"
META_FILE="$BOT_DIR/app.json"
REFRESHER_LOG="$BOT_DIR/refresher.log"
REFRESHER_PID_FILE="$BOT_DIR/refresher.pid"

cat > "$META_FILE" <<JSON
{
  "bot_id": "$BOT_ID",
  "app_id": $APP_ID,
  "app_name": "$APP_NAME",
  "repo": "$REPO",
  "pem_path": "$PEM_PATH",
  "repo_root": "$REPO_ROOT",
  "adapter": "$ADAPTER",
  "configured_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
JSON
chmod 600 "$META_FILE"
ok "metadata: $META_FILE"

# ── Mint initial token ───────────────────────────────────────────────
echo "Minting installation token..."
if ! TOKEN=$(mint_installation_token "$PEM_PATH" "$APP_ID"); then
  echo
  err "Token mint failed."
  err "Most common cause: the App isn't installed on $REPO yet (Step 4)."
  err "Go back to https://github.com/settings/apps/${APP_NAME}, install on $REPO,"
  err "then re-run this script."
  exit 1
fi
atomic_write "$TOKEN_FILE" "$TOKEN"
ok "token minted + saved: $TOKEN_FILE"

# ── Install adapter's gh wrapper (bot-scoped) ────────────────────────
adapter_install_gh_wrapper "$BOT_DIR"
ok "gh wrapper installed: $BOT_DIR/bin/gh"

# ── Restart refresher ────────────────────────────────────────────────
if [[ -f "$REFRESHER_PID_FILE" ]] && kill -0 "$(<"$REFRESHER_PID_FILE")" 2>/dev/null; then
  warn "Killing prior refresher PID $(<"$REFRESHER_PID_FILE")"
  kill "$(<"$REFRESHER_PID_FILE")" 2>/dev/null || true
fi
: > "$REFRESHER_LOG"

# Refresher rotates the token every 40min. Runs detached via nohup; exits
# cleanly when the harness session is gone (so it doesn't outlive the agent).
nohup bash -c "
  set -uo pipefail
  cd / # ensure relative SCRIPT_DIR doesn't matter inside the loop
  source '$SCRIPT_DIR/_lib/mint-token.sh'
  pem='$PEM_PATH'; app='$APP_ID'; bot_id='$BOT_ID'
  token_file='$TOKEN_FILE'; log='$REFRESHER_LOG'
  echo \"\$(date -u +%FT%TZ) refresher: starting (run-bot-refresher-\${bot_id})\" >> \"\$log\"
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
    # Trim log if it grows past 1MB (keeps last 200 lines).
    if [[ -f \"\$log\" ]] && (( \$(stat -c %s \"\$log\" 2>/dev/null || echo 0) > 1048576 )); then
      tail -200 \"\$log\" > \"\$log.tmp\" && mv \"\$log.tmp\" \"\$log\"
    fi
  done
" >>"$REFRESHER_LOG" 2>&1 &
REFRESHER_PID=$!
echo "$REFRESHER_PID" > "$REFRESHER_PID_FILE"
disown "$REFRESHER_PID" 2>/dev/null || true
ok "token refresher running (PID $REFRESHER_PID, refreshes every 40min)"

# ── Restart harness via adapter ──────────────────────────────────────
if adapter_is_running "$BOT_ID"; then
  warn "Killing existing harness session: $BOT_ID"
  adapter_kill "$BOT_ID"
fi

adapter_launch "$BOT_ID" "$REPO_ROOT" "$TOKEN_FILE"
ok "harness launched: $BOT_ID"

# ── Done ─────────────────────────────────────────────────────────────
echo
cat <<DONE
${BOLD}${GREEN}━━━ Done. Your agent is live. ━━━${RESET}

  Bot:           ${BOT_ID}
  App:           ${APP_NAME} (id ${APP_ID})
  Repo:          ${REPO}
  Working dir:   ${REPO_ROOT}
  Bot files:     ${BOT_DIR}
  Adapter:       $(basename "$ADAPTER")
  Refresher PID: ${REFRESHER_PID}  (log: ${REFRESHER_LOG})

${BOLD}Re-run this script later:${RESET}
  Safe — same bot-id will kill + restart cleanly.
  Saved metadata: ${META_FILE}

${BOLD}Verify or repair without full restart:${RESET}
  bash $(dirname "$0" | sed "s|$HOME|~|")/verify-bot.sh ${BOT_ID}

${BOLD}Force a fresh token (after App permission changes):${RESET}
  bash $(dirname "$0" | sed "s|$HOME|~|")/remint-bot.sh ${BOT_ID}

${BOLD}Tear down:${RESET}
  kill ${REFRESHER_PID}
  rm -rf ${BOT_DIR}

DONE
