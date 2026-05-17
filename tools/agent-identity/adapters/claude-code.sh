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

# Adapter: Claude Code in a tmux session.
#
# Implements the harness adapter contract documented in ../README.md:
#   - adapter_check_prereqs              (called by run-bot.sh)
#   - adapter_install_gh_wrapper <bot-dir>
#   - adapter_launch <bot-id> <repo-root> <token-file>
#   - adapter_is_running <bot-id>
#   - adapter_kill <bot-id>
#
# Sourced by the lifecycle scripts (run-bot.sh / verify-bot.sh). Not meant
# to be executed directly.
#
# Replace this file (or set ADAPTER=<path> in the environment before running
# the lifecycle scripts) to plug in a different harness — e.g. gemini-cli,
# or a GKE sandbox-based harness.

# Required commands for the Claude Code adapter.
ADAPTER_REQUIRED_CMDS=(tmux claude)

adapter_check_prereqs() {
  local missing=()
  local cmd
  for cmd in "${ADAPTER_REQUIRED_CMDS[@]}"; do
    if ! command -v "$cmd" &>/dev/null; then
      missing+=("$cmd")
    fi
  done
  if (( ${#missing[@]} > 0 )); then
    echo "adapter[claude-code]: missing commands: ${missing[*]}" >&2
    echo "  Install tmux:    sudo apt install tmux" >&2
    echo "  Install claude:  npm install -g @anthropic-ai/claude-code" >&2
    return 1
  fi
}

# Install a per-bot `gh` wrapper that injects the token from $BOT_TOKEN_FILE
# on every invocation. Scoped to the bot's own bin/ — added to PATH inside
# the bot's tmux session, not installed user-globally.
adapter_install_gh_wrapper() {
  local bot_dir="$1"
  local bin_dir="$bot_dir/bin"
  local target="$bin_dir/gh"

  mkdir -p "$bin_dir"
  cat > "$target" <<'WRAPPER'
#!/bin/bash
# bot-gh-wrapper marker
set -euo pipefail
REAL_GH=""
for c in /usr/bin/gh /usr/local/bin/gh /opt/homebrew/bin/gh /snap/bin/gh; do
  [[ -x "$c" ]] && { REAL_GH="$c"; break; }
done
if [[ -z "$REAL_GH" || ! -x "$REAL_GH" ]]; then
  echo "bot-gh-wrapper: no real gh binary found on system" >&2; exit 127
fi
if [[ -z "${BOT_TOKEN_FILE:-}" ]]; then exec "$REAL_GH" "$@"; fi
[[ ! -f "$BOT_TOKEN_FILE" ]] && {
  echo "bot-gh-wrapper: token file missing: $BOT_TOKEN_FILE" >&2; exit 1; }
TOKEN=$(<"$BOT_TOKEN_FILE")
[[ -z "$TOKEN" ]] && {
  echo "bot-gh-wrapper: empty token in $BOT_TOKEN_FILE" >&2; exit 1; }
export GH_TOKEN="$TOKEN" GITHUB_TOKEN="$TOKEN"
exec "$REAL_GH" "$@"
WRAPPER
  chmod 755 "$target"
}

# Launch the harness in a fresh tmux session.
# The session inherits BOT_TOKEN_FILE and prepends the per-bot bin/ to PATH
# so every gh invocation inside the session reads the rotating token.
#
# --dangerously-skip-permissions is required for unattended operation in a
# tmux session. The agent will execute tool calls without prompting; the
# scoped GitHub App token plus the contained tmux scope mitigate the blast
# radius. See the security section of ../README.md for caveats.
adapter_launch() {
  local bot_id="$1" repo_root="$2" token_file="$3"
  local bot_dir token

  bot_dir=$(dirname "$token_file")
  token=$(<"$token_file")

  tmux new-session -d -s "$bot_id" -c "$repo_root" \
    -e "BOT_TOKEN_FILE=$token_file" \
    -e "GH_TOKEN=$token" \
    -e "GITHUB_TOKEN=$token" \
    -e "PATH=$bot_dir/bin:$PATH"

  tmux send-keys -t "$bot_id" \
    "claude --model opus --dangerously-skip-permissions" C-m
}

adapter_is_running() {
  local bot_id="$1"
  tmux has-session -t "$bot_id" 2>/dev/null
}

adapter_kill() {
  local bot_id="$1"
  if tmux has-session -t "$bot_id" 2>/dev/null; then
    tmux kill-session -t "$bot_id"
  fi
}

# A short human-readable description shown in script output.
adapter_describe() {
  echo "Claude Code in tmux (claude --model opus --dangerously-skip-permissions)"
}
