# discord-mcp

A single Go binary with two jobs against a Discord guild you administer:

- **`mine`** walks a guild's full history — channels, threads (including archived and private), and forum posts — into grep-friendly markdown files on disk, with resumable cursors and a per-run audit report.
- **`serve`** exposes read-only, live-fetch [MCP](https://modelcontextprotocol.io) tools over Streamable HTTP, so an AI agent can expand anything it finds in the archive into current Discord context.

No database. No gateway connection. No write path to Discord, ever. Server-agnostic: point it at any guild your bot is in.

## Why markdown, not a database

Discord's message-search endpoint is a user-account API — bots get a 403 on it. There is no bot-accessible full-text search over guild history at any price, so `search_messages` isn't a tool this project could honestly ship.

Instead, `mine` writes the guild's history as markdown on the same filesystem an agent already reads and greps. Discovery happens with `grep`/`Read`, the agent's native retrieval mode; the MCP tools are for the follow-up — "expand this hit," "is this still current," "who said this" — not for search. The archive is also plain text: it's `git`-able if you choose to track it, diff-able, and requires no infrastructure beyond a directory on disk.

## Setup

### 1. Create a Discord application and bot

1. [Discord Developer Portal](https://discord.com/developers/applications) → **New Application** → **Bot** → **Reset Token**. Save the token — it's the bot's sole credential and the only thing `discord-mcp` needs to talk to Discord.
2. **Enable the Message Content privileged intent.** Bot → Privileged Gateway Intents → **Message Content Intent**. This is not optional and easy to miss: without it, Discord returns messages with silently empty `content`, `embeds`, and `attachments` for every message the bot didn't author — no error, just blanks. `mine` samples the first batch of the first channel it reads and **aborts the entire run** if every sampled message came back blank, naming this exact step, rather than writing an archive full of nothing. Under 100 guilds this intent self-enables with no verification step.
3. **Install the bot to your guild** with, at minimum:
   - `View Channels`
   - `Read Message History`

   OAuth2 URL Generator → scope `bot` → those permissions → install.
4. **Optionally grant `Manage Threads`** if you want private archived threads mined. Without it, `GET /channels/{id}/threads/archived/private` returns 403 and private threads the bot hasn't joined are invisible to it — `mine` records this as a capability gap per channel in the run report, and still archives everything else (public threads, regular messages) normally.
5. **Audit per-channel permission overwrites.** A guild-level grant can be overridden by a channel-level deny. Any channel that denies the bot `View Channels` is invisible to `mine`, and the archive is silently incomplete unless you check the run report — every denied channel is recorded there by ID and name, never skipped quietly.

`Read Message History` covers the entire backlog; there's no "since the bot joined" limit.

### 2. Build

```bash
make build
```

Produces `bin/discord-mcp`. Requires Go 1.26.

## Configuration

Both subcommands read configuration from the environment; flags override where noted. Missing required values cause the process to log the problem and exit non-zero before doing anything else.

| Variable | Required | Default | Meaning |
| --- | --- | --- | --- |
| `DISCORD_BOT_TOKEN` | yes | — | The bot token from step 1. Env only — never pass it as a flag, put it in a config file, or check it into anything. |
| `DISCORD_GUILD_ID` | yes | — | The one guild this instance operates on. Every tool call and every mined message is scoped to it. |
| `PORT` | no | `127.0.0.1:9091` | `serve`'s listen address. `--port` overrides. |
| `DISCORD_MCP_AUTH_TOKEN` | no | — | Static bearer token required on `/mcp` requests. Required if `PORT` binds anything other than loopback — see below. |
| `DISCORD_ARCHIVE_DIR` | no | `../discord-archive` | Where `mine` writes the archive, relative to the working directory. |
| `DISCORD_MINE_MIN_INTERVAL` | no | `250ms` | Floor delay between outbound Discord REST calls during `mine`. Accepts any `time.ParseDuration` string. |

**The listen address fails closed.** `serve` defaults to binding loopback only (`127.0.0.1:9091`), which needs no inbound credential — that matches the trust model of a local MCP client config. If you point `PORT` at a non-loopback address (e.g. `0.0.0.0:9091`) without setting `DISCORD_MCP_AUTH_TOKEN`, `serve` refuses to bind and exits non-zero naming the missing variable, rather than opening an unauthenticated relay onto a guild's message history. When the token is set, `/mcp` requires `Authorization: Bearer <token>`; a missing or wrong bearer gets a 401 with a `WWW-Authenticate` challenge.

## Usage: `mine`

```bash
discord-mcp mine                                  # all mineable channels, resume from the archive
discord-mcp mine --channel 123456789012345678     # restrict the run to one channel
discord-mcp mine --from 987654321098765432         # repair: restart --channel's walk at this message ID
discord-mcp mine --dry-run                        # enumerate and report the plan, write nothing
```

**Resume is automatic.** For each channel, the resume cursor is derived from the newest `last_message_id` across that channel's existing window files — re-running `mine` with no flags picks up where the last run left off rather than re-reading history. `--from` overrides the derived cursor for a single `--channel`, for repairing a gap. `--dry-run` runs the full enumeration and produces a plan/report without touching the archive.

`SIGINT` (Ctrl-C) flushes and closes the currently open window file with its cursor before exiting, so an interrupted run resumes cleanly rather than re-reading a month of messages.

### Rate-limit behavior

`mine` talks to Discord strictly sequentially — one request in flight, ever — with a **250ms floor delay** (`DISCORD_MINE_MIN_INTERVAL`) between every call regardless of response headers. On top of the floor:

- If a response reports `X-RateLimit-Remaining: 0`, the client sleeps `X-RateLimit-Reset-After` plus jitter before its next call, rather than spending the last token and eating a 429.
- A 429 sleeps `retry_after` plus jitter and retries the same request, up to 5 attempts, before recording that channel as failed and moving to the next.
- A global 429 halves the request rate **for the remainder of the run** and never restores it.
- A 401 (bad or revoked token) aborts the entire run immediately — it's never retried, because retrying 401s is exactly how a sustained-invalid-response IP ban gets earned.
- A 403 on a single channel is recorded as denied and the walk continues.

This is deliberately boring and non-parallel: at 100 messages per request and the 250ms floor, even 200,000 messages is under nine minutes of wall-clock time. Parallelism buys nothing measurable here and risks a ban.

## Usage: `serve`

```bash
discord-mcp serve
```

Starts the Streamable HTTP MCP server (default `127.0.0.1:9091`), plus `GET /healthz` for liveness checks. `serve` never writes to disk and never persists anything — every tool call is a live Discord REST fetch, rendered to curated text.

### MCP tools

| Tool | What it does |
| --- | --- |
| `list_channels` | Lists the guild's channels: names, IDs, types, categories. The entry point — nothing else can be called without a channel ID. |
| `list_threads` | Lists active and archived threads (including forum posts) under a channel. |
| `get_messages` | Pages through a channel or thread's messages with `before`/`after`/`around`/`limit`. |
| `get_context` | Expands a message into the conversation around it — the tool you reach for after a grep hit in the archive. |
| `get_thread` | Reads a thread or forum post's metadata plus its full message history, including archived threads. |
| `resolve_user` | Looks up a Discord user by ID, or searches by username, display name, or nickname. |

Every tool verifies the target channel or thread belongs to the configured `DISCORD_GUILD_ID` before returning anything — a channel ID from another guild returns a tool error, never data.

### Registering with Claude Code

`serve` doesn't get launched by the MCP client — you run the binary yourself (`make run`, a systemd unit, a launchd agent, whatever fits). Register it as an HTTP server in `.mcp.json`:

```json
{
  "mcpServers": {
    "discord": {
      "type": "http",
      "url": "http://127.0.0.1:9091/mcp"
    }
  }
}
```

No secret goes in `.mcp.json` — the bot token reaches the server process via `DISCORD_BOT_TOKEN` in its own environment, never through the client config.

## Archive layout

```text
discord-archive/
├── guild.md                                  # guild + channel index, rebuilt each run
├── _runs/
│   └── 2026-08-20T14:02:11Z.md               # one audit report per mine run
└── channels/
    └── rules-questions-1122334455667788/     # <slug>-<channel_id>
        ├── _channel.md                       # channel metadata + window index, rebuilt each run
        ├── 2024-11.md
        ├── 2024-12.md
        ├── 2026-08.md                        # the open window; rewritten each run until it closes
        └── threads/
            └── rest-inheritance-1247891300000000000.md
```

Windows are calendar months (UTC), one file per channel per month, named `YYYY-MM.md`. Each window file carries YAML frontmatter with the resume cursor and metadata:

```yaml
---
guild_id: "1100000000000000000"
guild_name: My Guild
channel_id: "1122334455667788"
channel_name: rules-questions
channel_type: text
parent_category: Game
window: "2026-06"
window_start: 2026-06-01T00:00:00Z
window_end: 2026-07-01T00:00:00Z
timezone: UTC
first_message_id: "1247000000000000000"
last_message_id: "1249999999999999999"
message_count: 412
complete: true
mined_at: 2026-06-30T23:14:02Z
miner_version: 0.1.0
format: 1
---
```

A window's resume cursor is `max(last_message_id)` across all of a channel's window files. `complete: true` windows are immutable and never rewritten; only the current open window is rewritten on each run. `_channel.md` and `guild.md` are pure caches, rebuilt from window frontmatter on every run — deleting either loses nothing.

Threads get their own file under `threads/`, named `<slug>-<thread_id>.md`, with a pointer line rendered into the parent channel's window file at the position of the thread-creating message. Forum channels have no window files at all — a forum post *is* a thread, so a forum channel's entire content lives under `threads/`.

Every `mine` run writes one report to `_runs/<RFC3339-timestamp>.md`: channels attempted, channels denied (with IDs and names), threads-capability gaps, messages archived, windows opened/closed, threads archived, requests made, 429s absorbed, global-limit hits, and per-channel failures. This is the audit trail for what was read from the guild and when.

## Development

```bash
make build   # go build -o bin/discord-mcp ./cmd/discord-mcp
make test    # go test ./... -count=1
make lint    # golangci-lint run ./...
make fmt     # gofmt -w .
make vet     # go vet ./...
make run     # go run ./cmd/discord-mcp serve
make e2e     # scripts/e2e.sh — see below
```

The unit test suite (`make test`) runs against mocked Discord responses (`httptest`) and in-memory MCP sessions — no network access and no credentials needed.

The end-to-end suite (`test/e2e/e2e_test.go`, build-tagged `e2e`) drives the real compiled binary against a real Discord guild. It's excluded from `go test ./...` and `make test`, and requires `DISCORD_BOT_TOKEN` and `DISCORD_GUILD_ID` in the environment before running `make e2e`.

## License

MIT.
