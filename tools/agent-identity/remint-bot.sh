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

# Force-remint an agent identity's installation token, ignoring freshness.
#
# Usage:
#   bash remint-bot.sh <bot-id> [--help]
#
# What it does:
#   1. Reads saved metadata from ~/.bots/<bot-id>/app.json
#   2. Mints a NEW installation token via the App's .pem (no cache check).
#   3. Atomically overwrites ~/.bots/<bot-id>/token with the new token.
#   4. Prints token prefix + the App identity for verification.
#
# When to use:
#   - You just updated the App's permissions in the GitHub UI and want
#     the bot to pick up the new scopes immediately, not wait up to
#     40min for the refresher loop's next tick.
#   - You suspect the token is corrupt / truncated.
#
# Contrast with verify-bot.sh: that script is verify-or-repair (only
# re-mints if stale or missing). This script is force-remint (always
# mints fresh, even if the existing token is fresh).
#
# Does NOT touch the refresher loop or the harness session — only the
# token file. A running harness picks up the new token on its next gh
# invocation (the wrapper reads the file each call).

set -euo pipefail

case "${1:-}" in
  -h|--help)
    cat <<'HELP'
Force-remint an agent identity's installation token, ignoring freshness.

Usage:
  bash remint-bot.sh <bot-id> [--help]

What it does:
  1. Reads saved metadata from ~/.bots/<bot-id>/app.json
  2. Mints a NEW installation token via the App's .pem (no cache check).
  3. Atomically overwrites ~/.bots/<bot-id>/token with the new token.
  4. Prints token prefix + the App identity for verification.

When to use:
  - You just updated the App's permissions in the GitHub UI and want
    the bot to pick up the new scopes immediately, not wait up to
    40min for the refresher loop's next tick.
  - You suspect the token is corrupt / truncated.

Contrast with verify-bot.sh: that script is verify-or-repair (only
re-mints if stale or missing). This script is force-remint (always
mints fresh, even if the existing token is fresh).

Does NOT touch the refresher loop or the harness session — only the
token file. A running harness picks up the new token on its next gh
invocation (the wrapper reads the file each call).
HELP
    exit 0
    ;;
esac

if [[ $# -ne 1 ]]; then
  cat >&2 <<USAGE
Usage: $(basename "$0") <bot-id>

Force-remints the installation token for an agent configured via
run-bot.sh. Always mints fresh — does not check token age.

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

BOT_DIR="$HOME/.bots/$BOT_ID"
META_FILE="$BOT_DIR/app.json"
TOKEN_FILE="$BOT_DIR/token"

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
APP_NAME=$(read_meta app_name)

if [[ -z "$APP_ID" || -z "$PEM_PATH" ]]; then
  err "Saved metadata is incomplete: $META_FILE"
  err "Required: app_id, pem_path. Re-run run-bot.sh to refresh."
  exit 1
fi
if [[ ! -f "$PEM_PATH" ]]; then
  err "Saved pem_path is no longer a file: $PEM_PATH"
  err "The .pem may have been moved/deleted. Re-run run-bot.sh."
  exit 1
fi

step "Force re-minting token: $BOT_ID"
note "  app_id    = $APP_ID"
note "  app_name  = $APP_NAME"
note "  pem       = $PEM_PATH"
note "  token     = $TOKEN_FILE"

if [[ -f "$TOKEN_FILE" && -s "$TOKEN_FILE" ]]; then
  TOKEN_AGE=$(( $(date +%s) - $(stat -c %Y "$TOKEN_FILE") ))
  note "  current token age: ${TOKEN_AGE}s — will overwrite"
fi

step "Minting"

if ! TOKEN=$(mint_installation_token "$PEM_PATH" "$APP_ID"); then
  err "Token mint failed."
  err "Most common causes:"
  err "  - App not installed on the repo"
  err "  - .pem moved/corrupted at: $PEM_PATH"
  err "  - Pending permission updates need owner approval at:"
  err "    https://github.com/settings/apps/${APP_NAME}/installations"
  exit 1
fi

mkdir -p "$BOT_DIR"
atomic_write "$TOKEN_FILE" "$TOKEN"
ok "Token minted + saved: $TOKEN_FILE"
note "  prefix: ${TOKEN:0:12}... (${#TOKEN} chars)"

step "Verifying token"

IDENT=$(curl -sf -H "Authorization: token $TOKEN" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/installation/repositories" 2>/dev/null || echo "")

if [[ -z "$IDENT" ]]; then
  warn "Could not query /installation/repositories — token may still be propagating."
  note "  Try: curl -H \"Authorization: token \$(<$TOKEN_FILE)\" https://api.github.com/installation/repositories"
else
  REPO_COUNT=$(printf '%s' "$IDENT" \
    | python3 -c "import json,sys; print(json.load(sys.stdin).get('total_count', '?'))" 2>/dev/null \
    || echo "?")
  ok "Token valid — sees $REPO_COUNT repo(s)"
fi

echo
cat <<DONE
${BOLD}${GREEN}━━━ Re-mint complete. ━━━${RESET}

  Bot:        $BOT_ID
  App:        $APP_NAME (id $APP_ID)
  Token file: $TOKEN_FILE  (${#TOKEN} chars)

${BOLD}A running harness picks up the new token on its next gh${RESET}
${BOLD}invocation${RESET} (the wrapper reads the file per call).
No harness restart needed.

${BOLD}If you just updated App permissions:${RESET}
The new token carries the current App scopes. If a permission still
returns 403, check that the install owner accepted any pending updates
at https://github.com/settings/apps/${APP_NAME}/installations

DONE
