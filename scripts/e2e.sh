#!/bin/sh
# End-to-end acceptance: the real discord-mcp binary against a real
# Discord guild. Requires DISCORD_BOT_TOKEN and DISCORD_GUILD_ID in
# the environment (see .env.example). Kept out of `go test ./...` behind
# the e2e build tag; run explicitly via `make e2e`.
set -eu

echo "==> building discord-mcp"
go build -o /tmp/discord-mcp-e2e ./cmd/discord-mcp

echo "==> running e2e acceptance tests"
go test -tags e2e ./test/e2e/... -count=1 -v

echo "==> e2e PASSED"
