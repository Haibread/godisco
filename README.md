# godisco

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
- Periodic rename loop that keeps secondary channel names in sync with the
  current activity / game (every 5 minutes — Discord rate limit).
- Cleanup at startup: channels emptied or deleted while the bot was offline
  are removed.
- SQLite-backed state via GORM, persisted in `config/channels.db`.
- Slash commands for runtime configuration (`/ping`, `/help`,
  `/create-primary`).

See [ARCHITECTURE.md](ARCHITECTURE.md) for the design and the reasoning behind
it.

## Getting started

### Prerequisites

- A Discord bot token, with the `GUILD_PRESENCES` privileged intent enabled in
  the developer portal
- Docker, or Go 1.24+ and a C compiler to build from source

### Configuration

godisco reads `config/config.yaml` via [viper] and watches the file for
changes.

```yaml
# config/config.yaml
token: "YOUR_DISCORD_BOT_TOKEN"
bot_status: ""
```

| Key          | Default | Description                              |
| ------------ | ------- | ---------------------------------------- |
| `token`      | `""`    | Discord bot token (required)             |
| `bot_status` | `""`    | "Listening to ..." status string         |

The SQLite database is created automatically at `config/channels.db` on first
run.

godisco subscribes to these gateway intents:

- `GUILDS`
- `GUILD_MESSAGES`
- `GUILD_VOICE_STATES`
- `GUILD_PRESENCES` (used to detect the active game for `{{.GameName}}`)

### Usage

With Docker:

```sh
mkdir -p config
echo 'token: "YOUR_TOKEN"' > config/config.yaml
docker compose up -d
```

The pre-built image is published to GitHub Packages on every push to `main`:

```
ghcr.io/haibread/godisco:latest
```

From source:

```sh
go build -o godisco ./app/godisco
mkdir -p config
echo 'token: "YOUR_TOKEN"' > config/config.yaml
./godisco
```

## Slash commands

| Command              | Permissions       | Description                                                |
| -------------------- | ----------------- | ---------------------------------------------------------- |
| `/ping`              | anyone            | Reply with command delay and gateway heartbeat latency.    |
| `/help`              | anyone            | Placeholder.                                               |
| `/create-primary`    | `Manage Channels` | Create a new primary voice channel (see below).            |

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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Apache 2.0 — see [LICENSE](LICENSE).

[viper]: https://github.com/spf13/viper
[text/template]: https://pkg.go.dev/text/template
