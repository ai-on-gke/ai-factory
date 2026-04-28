# Agent Identity

Per-agent GitHub App identities for ai-factory.

## Why per-agent identity

ai-factory's agents (`speccer`, `planner`, `builder`, `reviewer`,
`top-level`, and others to come) act on GitHub: they open issues, push
branches, open PRs, leave review comments, merge changes. Today, all of
those actions flow through a single shared GitHub identity. As the agent
roster grows, the single-identity model has limits:

- **No per-agent audit trail.** Every action looks like the same actor.
  When you grep `git log` or PR history for "what did the reviewer do
  yesterday", you can't tell.
- **Shared rate-limit and abuse-detection budget.** A misbehaving agent
  throttles every other agent on the same identity.
- **All-or-nothing permission scoping.** A reviewer agent that should
  only need `pull-requests: read` ends up with the same scopes as a
  builder agent that needs broad write.
- **Single point of compromise.** One leaked token compromises every
  agent.

The fix is to give each agent its own GitHub App. Each App has its own
private key, its own installation, its own short-lived (1h)
installation token, and its own scoped permissions. The scripts in this
directory provision and rotate those identities.

## What's here

- `run-bot.sh` — interactive walkthrough that creates the App in your
  browser, downloads the private key, installs the App on a target
  repo, captures the App ID, mints the first installation token, and
  launches the configured harness adapter. Run once per new agent.
- `verify-bot.sh <bot-id>` — idempotent verify-or-repair: re-mints
  stale tokens, restarts dead refresher loops, leaves working pieces
  alone. Safe to run repeatedly (e.g. from a watchdog).
- `remint-bot.sh <bot-id>` — force-remint the token immediately. Use
  after editing the App's permissions in the GitHub UI when you don't
  want to wait for the refresher loop's next tick.
- `_lib/mint-token.sh` — shared library: JWT signing, installation
  token minting, atomic file writes. Sourced by all three scripts.
- `adapters/claude-code.sh` — reference harness adapter (see "Harness
  adapters" below).
- `adapters/README.md` — how to write a new adapter.

## Lifecycle

```
First time a new agent comes online:

    operator runs        run-bot.sh
                              │
                              ▼
                  ┌───────────────────────┐
                  │ ~/.bots/<bot-id>/     │
                  │   app.json   (meta)   │
                  │   token      (mode 600)│
                  │   bin/gh     (wrapper)│
                  │   refresher.{pid,log} │
                  └───────────────────────┘
                              │
                              ▼
                       harness session
                       (adapter-specific)


Every 40 minutes, while the harness is alive:

    refresher loop  ──►  mint new token  ──►  atomic_write to token file
                              │
                              ▼
                    harness's next gh call
                    reads the file, picks up
                    the new token transparently


Operator drops in occasionally:

    bash verify-bot.sh <bot-id>   # idempotent — fixes only what's broken
    bash remint-bot.sh <bot-id>   # force-mint, e.g. after permission change
```

The 40-minute refresh cadence sits comfortably below the 1-hour token
TTL and well above any reasonable burst of activity.

## Harness adapter contract

The lifecycle scripts are agent-runtime-agnostic. They source an
**adapter** that knows how to launch the agent harness, install a
per-call token wrapper, and check whether the harness is running. Any
agent runtime that satisfies the contract below can be plugged in.

An adapter is a bash file that defines these functions:

| Function | Called by | Purpose |
|---|---|---|
| `adapter_check_prereqs` | `run-bot.sh` | Verify the harness's binary dependencies are installed. Return non-zero if not. |
| `adapter_install_gh_wrapper <bot-dir>` | `run-bot.sh` | Install a `gh` wrapper at `<bot-dir>/bin/gh` that reads the token from `$BOT_TOKEN_FILE` on every invocation. The wrapper is bot-scoped, not user-global. |
| `adapter_launch <bot-id> <repo-root> <token-file>` | `run-bot.sh`, `verify-bot.sh` | Start the harness. Must put `<bot-dir>/bin` on `$PATH` and set `BOT_TOKEN_FILE=<token-file>` in the environment. |
| `adapter_is_running <bot-id>` | `verify-bot.sh` | Return 0 if the harness session is alive, non-zero otherwise. |
| `adapter_kill <bot-id>` | `run-bot.sh` | Stop the harness session (called before relaunch on re-run). |
| `adapter_describe` | `run-bot.sh` | Print a one-line human-readable description, used in script output. |

The contract assumes the harness:

- Reads the token from `$BOT_TOKEN_FILE` **on every `gh` / `git` call**,
  not once at startup. The included `adapters/claude-code.sh` provides
  a `gh` wrapper script that satisfies this; another adapter could
  satisfy it via a sidecar that mounts a Kubernetes Secret, etc.
- Sets the GitHub-side commit author to the bot's noreply identity
  (`<app-id>+<app-name>[bot]@users.noreply.github.com`) before
  committing, so commits are attributed to the App account.

The default adapter is `adapters/claude-code.sh` (Claude Code in a
tmux session). Override with `ADAPTER=/path/to/adapter.sh` in the
environment. The chosen adapter is recorded in `~/.bots/<bot-id>/app.json`
so subsequent `verify-bot.sh` runs use the same one.

See `adapters/README.md` for instructions on writing new adapters.

## Files on disk

After `run-bot.sh` finishes, an agent's state lives entirely under
`~/.bots/<bot-id>/`:

```
~/.bots/<bot-id>/
├── app.json            # configured_at, app_id, pem_path, repo, repo_root, adapter
├── token               # current installation token (mode 600)
├── bin/
│   └── gh              # adapter-installed gh wrapper (mode 755)
├── refresher.log       # rotating log; auto-trimmed at 1MB
└── refresher.pid       # PID of the refresh loop
```

`~/.bots/` and each `~/.bots/<bot-id>/` are mode 700. The token and
metadata files are mode 600. Adapters write the harness binary's wrapper
to `bin/gh` mode 755.

## Tear-down

```bash
kill $(cat ~/.bots/<bot-id>/refresher.pid) 2>/dev/null
# Adapter-specific harness-stop, e.g. for the Claude Code adapter:
tmux kill-session -t <bot-id>
rm -rf ~/.bots/<bot-id>
```

You should also delete the GitHub App at
`https://github.com/settings/apps/<app-name>/advanced` once the agent
is permanently retired.

## Security considerations

- **Private keys.** The `.pem` is an RSA private key for the App. Anyone
  with this file can mint installation tokens. The scripts leave it
  wherever you originally downloaded it (typically `~/Downloads/`); move
  it somewhere appropriate before granting access to the host.
- **Token file.** Mode 600. A torn read is impossible because tokens
  are written atomically (write to a tempfile in the same directory,
  then `mv` over the target).
- **Refresher loop.** Runs detached via `nohup`. Exits on its own when
  the harness session is gone, so it doesn't outlive the agent.
- **GitHub App permissions.** Set them as narrowly as the agent's job
  allows. A reviewer that only needs to read code and post review
  comments doesn't need write access to repository contents.
- **Harness sandboxing.** This depends on the adapter. The default
  Claude Code adapter runs the harness with
  `--dangerously-skip-permissions` so it can act without prompting in
  an unattended tmux session. The scoped GitHub App token plus the
  contained tmux scope mitigate the blast radius — the agent cannot do
  more than the App's permissions allow on GitHub — but anything in the
  agent's prompt context can take action against your local system. A
  GKE-sandbox adapter (future work) would isolate that further by
  running the harness inside an `agent-sandbox` Pod.

## Future work

- **`gemini-cli` adapter** — `gemini-cli` is the agent runtime named in
  the top-level README, but ai-factory's agent runtime is in flux (see
  issue #43). A `gemini-cli` adapter is intended once that direction
  settles.
- **GKE-sandbox adapter** — for ai-factory's actual deployment target
  (GKE with `agent-sandbox`), the adapter would launch the harness
  inside a `Sandbox` Pod, mount the token file via a `Secret`, and use
  a sidecar to rotate the secret instead of a per-host refresher loop.
- **Non-interactive provisioning mode for `run-bot.sh`** — the current
  flow is human-driven (operator points-and-clicks in the GitHub UI for
  App creation). For automated provisioning at scale, this would be
  replaced with API-driven App + installation creation.
- **Multi-installation Apps** — `_lib/mint-token.sh` accepts an explicit
  installation ID, but the lifecycle scripts always use the App's first
  installation. Lifting that limitation matters once an App is
  installed in multiple owners or orgs.

## Related

- ai-factory's `AGENTS.md` — agent definitions and conventions.
- ai-factory's `tools/run-subagent` — runs an agent inside a Sandbox
  Pod. Today it uses a shared identity; an integration with this
  scheme is the GKE-sandbox adapter described in "Future work".
