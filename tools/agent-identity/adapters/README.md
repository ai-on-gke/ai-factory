# Adapters

A harness adapter teaches `run-bot.sh` and `verify-bot.sh` how to
launch and manage a specific agent runtime. The lifecycle scripts —
JWT signing, installation token minting, refresher loop, atomic file
writes — are the same regardless of harness; only the bits that touch
the harness itself are adapter-specific.

## Writing a new adapter

An adapter is a single bash file that defines six functions. See
`claude-code.sh` for a reference implementation.

### Required functions

```bash
# Verify the harness's binary dependencies are installed.
# Print any missing dependencies (with install hints) to stderr.
# Return non-zero on missing prereqs.
adapter_check_prereqs

# Install a `gh` wrapper at <bot-dir>/bin/gh that reads the token from
# $BOT_TOKEN_FILE on every invocation. Bot-scoped (not user-global) so
# multiple bots on the same host don't conflict.
adapter_install_gh_wrapper <bot-dir>

# Start the harness session.
# Must put <bot-dir>/bin on $PATH and set BOT_TOKEN_FILE=<token-file>
# in the environment so the gh wrapper picks up the rotating token.
adapter_launch <bot-id> <repo-root> <token-file>

# Return 0 if the harness session is alive, non-zero otherwise.
adapter_is_running <bot-id>

# Stop the harness session. Called before relaunch on re-run of run-bot.sh.
adapter_kill <bot-id>

# Print a one-line human-readable description of this adapter.
# Shown in script output during run-bot.sh.
adapter_describe
```

### Selecting an adapter

By default, `run-bot.sh` sources `adapters/claude-code.sh` from this
directory. Override by setting `ADAPTER=/path/to/your-adapter.sh` in
the environment before running:

```bash
ADAPTER=/path/to/my-adapter.sh bash run-bot.sh
```

The chosen adapter is recorded in `~/.bots/<bot-id>/app.json` so
subsequent `verify-bot.sh` runs reuse it without needing the env var.

### Contract assumptions

The adapter must satisfy these properties for the lifecycle to work:

1. **The harness reads the token file on every `gh` / `git` call**, not
   once at startup. The refresher loop rotates the token in place every
   ~40 minutes; if the harness caches it, calls will start failing
   ~1 hour in. The reference adapter satisfies this by installing a
   `gh` wrapper that re-reads `$BOT_TOKEN_FILE` per invocation.
2. **The harness sets the GitHub-side commit author to the bot's
   noreply identity** before committing, so commits are attributed to
   the App account in the GitHub UI:
   ```
   git config user.name  "<app-name>[bot]"
   git config user.email "<app-id>+<app-name>[bot]@users.noreply.github.com"
   ```
3. **`adapter_is_running` is cheap.** `verify-bot.sh` calls it on every
   invocation; an adapter that talks to a remote API for liveness
   should cache or short-circuit.

## Planned adapters (not yet implemented)

- **`gemini-cli`** — once ai-factory's choice of agent runtime
  stabilizes (see issue #43).
- **`gke-sandbox`** — launches the harness inside an `agent-sandbox`
  Pod on GKE, mounts the token via a Kubernetes `Secret`, and uses a
  sidecar to rotate the secret instead of a per-host refresher loop.
  This is the natural deployment target for ai-factory.
