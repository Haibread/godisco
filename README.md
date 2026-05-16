# godisco

A Discord bot that manages dynamic voice channels. When a user joins a
designated *primary* voice channel, godisco creates a *secondary* channel,
moves the user into it, and deletes the secondary channel automatically once
it becomes empty.

Secondary channel names are generated from a Go `text/template` string, with
variables for rank, ICAO phonetic alphabet, the active game, and the creator.

## Features

- Dynamic voice channels: join a primary channel, get your own secondary
  channel; leave it empty, it disappears.
- Templated channel names with variables (`{{.Icao}}`, `{{.Number}}`,
  `{{.GameName}}`, `{{.PartySize}}`, `{{.CreatorName}}`).
- Periodic rename loop that keeps secondary channel names in sync with the
  current activity / game (every 5 minutes — Discord rate limit).
- SQLite-backed state via GORM, persisted in `config/channels.db`.
- Slash commands for runtime configuration (`/ping`, `/help`,
  `/create-primary`).

## Architecture

```
app/godisco/   -> entrypoint: config, logger, DB, Discord session, handlers
channels/      -> voice-state handlers, name templates, rename loop
commands/      -> slash command registration and handlers
database/      -> GORM/SQLite setup and migrations
logging/       -> zap logger initialization
models/        -> GORM models (PrimaryChannel, SecondaryChannel)
```

Voice-state events fan out through `channels.VCUpdate`:

- user joined a channel -> if it's a primary, create a secondary and move them.
- user moved -> same logic on the new channel; if the old channel was a
  secondary that is now empty, delete it.
- user disconnected -> check if the previous secondary is empty, delete if so.

A background goroutine in `channels/loops.go` walks every secondary channel
every 5 minutes and renames it if its templated name has changed (e.g. the
group's game changed).

## Configuration

godisco reads `config/config.yaml` via [viper] and watches the file for
changes.

```yaml
# config/config.yaml
token: "YOUR_DISCORD_BOT_TOKEN"
bot_status: ""
bot_activity_type: "Listening"
```

| Key                 | Default       | Description                                                                  |
| ------------------- | ------------- | ---------------------------------------------------------------------------- |
| `token`             | `""`          | Discord bot token (required)                                                 |
| `bot_status`        | `""`          | Status string shown in the bot's profile. Empty disables status entirely.    |
| `bot_activity_type` | `"Listening"` | Activity verb: `Playing`, `Listening`, `Watching`, `Competing`, `Streaming`. |

The SQLite database is created automatically at `config/channels.db` on first
run.

### Required gateway intents

godisco subscribes to:

- `GUILDS`
- `GUILD_MESSAGES`
- `GUILD_VOICE_STATES`
- `GUILD_PRESENCES` (used to detect the active game for `{{.GameName}}`)

`GUILD_PRESENCES` is a privileged intent — enable it in the Discord developer
portal for the bot's application.

## Running

### Docker (recommended)

```sh
mkdir -p config
echo 'token: "YOUR_TOKEN"' > config/config.yaml
docker compose up -d
```

The pre-built image is published to GitHub Packages on every push to `main`:

```
ghcr.io/haibread/godisco:latest
```

### From source

Requires Go 1.24+.

```sh
go build -o godisco ./app/godisco
mkdir -p config
echo 'token: "YOUR_TOKEN"' > config/config.yaml
./godisco
```

## Slash commands

| Command              | Permissions       | Description                                                       |
| -------------------- | ----------------- | ----------------------------------------------------------------- |
| `/ping`              | anyone            | Reply with command delay and gateway heartbeat latency.           |
| `/help [topic]`      | anyone            | Show commands and template-field reference (filterable by topic). |
| `/create-primary`    | `Manage Channels` | Create a new primary voice channel (see below).                   |
| `/list-primaries`    | anyone            | List managed primary channels in the current server.              |
| `/delete-primary`    | `Manage Channels` | Remove a managed primary channel and its DB record.               |

All replies are ephemeral — only the user who ran the command sees them.

### `/create-primary`

Creates a new primary voice channel named `➕ New Channel`. Joining it will
spawn a secondary channel for the joining user.

Options:

- `default-name` *(required)* — fallback name used when the template renders
  empty (e.g. no game detected). Used as `#<rank> <default-name>`.
- `template` *(required)* — Go `text/template` string. See *Name templates*.

Both options are validated against fake data before the channel is created.

## Name templates

Templates are standard Go [text/template] syntax. The available fields are:

| Field           | Type   | Source                                                                |
| --------------- | ------ | --------------------------------------------------------------------- |
| `.Icao`         | string | NATO phonetic alphabet word for the channel rank (Alfa, Beta, ...).   |
| `.Number`       | string | Channel rank (1-based) among siblings of the same primary channel.    |
| `.GameName`     | string | Most common `Game`/`Competing` activity in the channel.               |
| `.PartySize`    | string | Currently always `"N/A"` (placeholder).                               |
| `.CreatorName`  | string | Username of the user who triggered the channel creation.              |

Examples:

```
{{.Icao}} - {{.GameName}}              -> "Alfa - Counter-Strike 2"
#{{.Number}} {{.CreatorName}}'s room   -> "#2 alice's room"
```

If `.GameName` cannot be resolved (no active game / missing presence), it
falls back to the primary channel's `default-name`, then to
`"Game Unknown"`. If the rendered template ends up empty, the channel name
falls back to `#<rank+1> <default-name>`.

The rename loop re-evaluates templates every 5 minutes per secondary channel
to track game changes.

## Troubleshooting

**`{{.GameName}}` always renders as the default name / "Game Unknown".**
Enable the privileged *Server Members* and *Presence* intents in the Discord
developer portal for the bot's application — without `GUILD_PRESENCES` the
gateway never sends activity data, so godisco can't see what anyone is
playing.

**Channel name doesn't update when someone changes game.**
Discord rate-limits channel renames to 2 per 10 minutes per channel. godisco
renames a channel at most once every 5 minutes — both the periodic sweep and
the presence-triggered rename go through the same throttle. If you just
changed game and the name didn't update, wait up to 5 minutes.

**`/create-primary` fails with "Failed to create channel: HTTP 403 Forbidden".**
The bot's role is missing `Manage Channels` in the guild (or in the category
where the channel would be created). Check the role's permissions and any
category-level overrides.

**Bot creates the secondary channel but doesn't move the user.**
The bot also needs `Move Members` to drag the user into the freshly-created
secondary. `Manage Channels` alone isn't enough.

**`/delete-primary` reports "primary still has N active secondary channel(s)".**
godisco refuses to delete a primary while there are live secondaries spawned
from it (deleting it would orphan them). Wait for the secondaries to empty
out — they auto-delete — then try again.

**`/create-primary` reports "Invalid `template`: ..."**.
The error includes the underlying Go `text/template` parse or execute
message. The most common case is referencing a field that doesn't exist
(e.g. `{{.Foo}}`); use `/help template` for the full field list.

## Development

```sh
go vet ./...
go test -race ./...
go build ./...
```

Tests cover the name template engine, voice-update handlers, and command
option parsing. The SQLite test DB is isolated per test to allow `-race`.

CI (`.github/workflows/`):

- `main-mr.yml` — vet + test on pull requests to `main`.
- `main.yml` — vet + test + Docker build/push to GHCR on pushes to `main`.
- `codeql-analysis.yml` — CodeQL scan on push, PR, and weekly schedule.

Dependencies are managed by [Renovate] (`.github/renovate.json`); Go modules
minor/patch and Docker base images are grouped, GitHub Actions are grouped.

## License

Apache 2.0 — see [LICENSE](LICENSE).

[viper]: https://github.com/spf13/viper
[text/template]: https://pkg.go.dev/text/template
[Renovate]: https://docs.renovatebot.com/
