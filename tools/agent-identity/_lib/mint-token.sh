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

# Shared library: mint a GitHub App installation token from a .pem + App ID.
#
# Sourced by run-bot.sh, verify-bot.sh, and remint-bot.sh. Not meant to be
# executed directly.
#
# Exposes:
#   mint_installation_token <pem-path> <app-id> [installation-id]
#     - On success: prints the token to stdout, returns 0.
#     - On failure: prints a diagnostic to stderr, returns non-zero.
#   atomic_write <path> <content>
#     - Writes content to a temp file in the same directory and renames over
#       <path>. Avoids torn reads from concurrent harness invocations.

# Build a 9-minute RS256 JWT signed by the App's .pem.
# (GitHub caps App JWTs at 10 minutes; we leave a minute of clock-skew slack.)
_jwt_for_app() {
  local pem="$1" app="$2"
  local now iat exp header payload h_b64 p_b64 signing_input sig

  now=$(date +%s); iat=$((now - 60)); exp=$((now + 540))
  header='{"alg":"RS256","typ":"JWT"}'
  payload="{\"iat\":${iat},\"exp\":${exp},\"iss\":\"${app}\"}"
  h_b64=$(printf '%s' "$header"  | openssl base64 -A | tr '+/' '-_' | tr -d '=')
  p_b64=$(printf '%s' "$payload" | openssl base64 -A | tr '+/' '-_' | tr -d '=')
  signing_input="${h_b64}.${p_b64}"
  sig=$(printf '%s' "$signing_input" \
    | openssl dgst -sha256 -sign "$pem" -binary 2>/dev/null \
    | openssl base64 -A | tr '+/' '-_' | tr -d '=')
  if [[ -z "$sig" ]]; then
    echo "mint-token: openssl signing failed (bad .pem at $pem?)" >&2
    return 1
  fi
  printf '%s.%s' "$signing_input" "$sig"
}

# Mint an installation access token.
# Args:
#   $1 — pem path
#   $2 — App ID (numeric)
#   $3 — installation ID (optional). If omitted, uses the App's first
#        installation. Pass an explicit ID when an App is installed in
#        multiple owners/orgs.
mint_installation_token() {
  local pem="$1" app="$2" installation_id="${3:-}"
  local jwt installs iid token_response token

  if [[ ! -f "$pem" ]]; then
    echo "mint-token: pem not found: $pem" >&2
    return 1
  fi
  if ! [[ "$app" =~ ^[0-9]+$ ]]; then
    echo "mint-token: app id must be numeric, got: $app" >&2
    return 1
  fi

  jwt=$(_jwt_for_app "$pem" "$app") || return 1

  if [[ -z "$installation_id" ]]; then
    installs=$(curl -sf -H "Authorization: Bearer $jwt" \
      -H "Accept: application/vnd.github+json" \
      "https://api.github.com/app/installations" 2>/dev/null || echo "")
    if [[ -z "$installs" || "$installs" == "[]" ]]; then
      echo "mint-token: App $app has no installations — install it on a repo first" >&2
      return 1
    fi
    iid=$(printf '%s' "$installs" \
      | python3 -c "import json,sys; print(json.load(sys.stdin)[0]['id'])" 2>/dev/null \
      || echo "")
    if [[ -z "$iid" ]]; then
      echo "mint-token: failed to parse installation id from /app/installations" >&2
      return 1
    fi
  else
    iid="$installation_id"
  fi

  token_response=$(curl -sf -X POST -H "Authorization: Bearer $jwt" \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/app/installations/${iid}/access_tokens" \
    2>/dev/null || echo "")
  if [[ -z "$token_response" ]]; then
    echo "mint-token: token POST failed (App permissions pending owner approval?)" >&2
    return 1
  fi
  token=$(printf '%s' "$token_response" \
    | python3 -c "import json,sys; print(json.load(sys.stdin)['token'])" 2>/dev/null \
    || echo "")
  if [[ -z "$token" ]]; then
    echo "mint-token: 'token' field missing in response" >&2
    return 1
  fi

  printf '%s' "$token"
}

# Atomic file write — write to a tempfile in the same directory, then rename.
# Renames are atomic on POSIX filesystems, so a concurrent reader either sees
# the old token or the new token, never a torn write.
atomic_write() {
  local path="$1" content="$2"
  local dir tmp
  dir=$(dirname "$path")
  tmp=$(mktemp "$dir/.tmp.XXXXXX") || { echo "atomic_write: mktemp failed" >&2; return 1; }
  printf '%s' "$content" > "$tmp" || { rm -f "$tmp"; return 1; }
  chmod 600 "$tmp" || { rm -f "$tmp"; return 1; }
  mv "$tmp" "$path" || { rm -f "$tmp"; return 1; }
}
