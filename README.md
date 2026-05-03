# godisco

[![quality](https://github.com/Haibread/godisco/actions/workflows/quality.yaml/badge.svg)](https://github.com/Haibread/godisco/actions/workflows/quality.yaml) [![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/Haibread/godisco/badge)](https://scorecard.dev/viewer/?uri=github.com/Haibread/godisco)

A Discord bot that manages dynamic voice channels. When a user joins a
designated *primary* voice channel, godisco creates a *secondary* channel,
moves the user into it, and deletes the secondary channel automatically once
it becomes empty.

## Description

Secondary channel names are generated from a Go `text/template` string, with
variables for rank, ICAO phonetic alphabet, the active game, and the creator.

- Dynamic voice channels: join a primary channel, get your own secondary
  channel; leave it empty, it disappears.
- Templated channel names with variables (`{{.Icao}}`, `{{.Number}}`,
  `{{.GameName}}`, `{{.PartySize}}`, `{{.CreatorName}}`).
- Secondary channel names follow the current activity / game: renamed when a
  member's game changes and swept every 5 minutes, at most one rename per
  channel every 5 minutes (Discord rate limit).
- Cleanup at startup: channels emptied or deleted while the bot was offline
  are removed.
- SQLite-backed state via GORM, persisted in `config/channels.db` by
  default.
- Slash commands to manage primary channels (see *Slash commands*).

See [ARCHITECTURE.md](ARCHITECTURE.md) for the design and the reasoning behind
it.

## Getting started

### Prerequisites

- A Discord bot token, with the `GUILD_PRESENCES` privileged intent enabled in
  the developer portal
- Docker, or Go 1.26+ to build from source

### Configuration

godisco reads `config/config.yaml` via [viper] and watches the file for
changes.

```yaml
# config/config.yaml
token: "YOUR_DISCORD_BOT_TOKEN"
bot_status: ""
bot_activity_type: "Listening"
```

Each setting comes from, in order of precedence, a command-line flag, an
environment variable, the config file, then its default.

| Key                 | Variable        | Flag            | Default              | Description                                                                 |
| ------------------- | --------------- | --------------- | -------------------- | --------------------------------------------------------------------------- |
| —                   | `CONFIG_FILE`   | `--config-file` | `config/config.yaml` | Config file to read                                                         |
| `token`             | `DISCORD_TOKEN` | —               | `""`                 | Discord bot token (required)                                                |
| `bot_status`        | `BOT_STATUS`    | —               | `""`                 | Status string shown in the bot's profile. Empty disables status entirely.   |
| `bot_activity_type` | —               | —               | `Listening`          | Activity verb: `Playing`, `Listening`, `Watching`, `Competing`, `Streaming` |
| `db_path`           | `DB_PATH`       | `--db-path`     | `config/channels.db` | SQLite database, created on first run                                       |

Relative paths resolve against the working directory, `/app` in the image,
where a moved database must sit in a mounted directory owned by UID 65532.
The config file is required, unless it is left at its default and
`DISCORD_TOKEN` is set. Paths and the token are read once at start-up.

godisco subscribes to these gateway intents:

- `GUILDS`
- `GUILD_MESSAGES`
- `GUILD_VOICE_STATES`
- `GUILD_PRESENCES` (used to detect the active game for `{{.GameName}}`)

### Usage

With Docker, using the image published to GitHub Packages
(`ghcr.io/haibread/godisco`). Each release publishes it under its version
(such as `1.2.0`), and a stable release also moves `latest`, which
[`docker-compose.yaml`](docker-compose.yaml) runs:

```sh
mkdir -p config
echo 'token: "YOUR_TOKEN"' > config/config.yaml
sudo chown -R 65532:65532 config
docker compose up -d
```

The container runs as UID/GID 65532, so `config/` and everything in it must be
owned by that user: the bot exits with `permission denied` on
`channels.db` otherwise.

Docker reports the container `unhealthy` when the bot stops receiving gateway
heartbeat acknowledgements from Discord.

To update a running instance to the latest stable release:

```sh
docker compose pull
```

```sh
docker compose up -d
```

A released image carries a build provenance attestation and a CycloneDX SBOM
attestation, and its GitHub release attaches the SBOM. To check that this
repository's CI built the image you run, with the
[GitHub CLI](https://cli.github.com/):

```sh
gh attestation verify oci://ghcr.io/haibread/godisco:latest --repo Haibread/godisco
```

The bot logs its version at startup.

From source:

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
| `.PartySize`    | string | Always `"N/A"` (placeholder).                                         |
| `.CreatorName`  | string | Username of the user who triggered the channel creation.              |

Examples:

```
{{.Icao}} - {{.GameName}}              -> "Alfa - Counter-Strike 2"
#{{.Number}} {{.CreatorName}}'s room   -> "#2 alice's room"
```

If `.GameName` cannot be resolved (no active game / missing presence), it
falls back to the primary channel's `default-name`, then to
`"Game Unknown"`. If the rendered template ends up empty, the channel name
falls back to `#<rank> <default-name>`.

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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Apache 2.0 — see [LICENSE](LICENSE).

[viper]: https://github.com/spf13/viper
[text/template]: https://pkg.go.dev/text/template
