# Contributing

Open an issue or a pull request on
[GitHub](https://github.com/Haibread/godisco). A change lands once the checks
below are green.

## Development setup

- [Go](https://go.dev/doc/install) 1.24 or newer
- A C compiler for cgo, which the SQLite driver needs: `gcc` or `clang`
- [pre-commit](https://pre-commit.com/#install)
- [golangci-lint](https://golangci-lint.run/docs/welcome/install/), to reproduce
  the lint job locally

```bash
git clone https://github.com/Haibread/godisco.git
```

```bash
pre-commit install
```

`go build ./...` succeeding means the setup works. See the
[README](README.md#getting-started) for running the bot.

## Running the tests

The whole suite, which must pass before a pull request:

```bash
go test -race ./...
```

A single test:

```bash
go test -race ./channels/ -run TestReconcile_MissingChannelDropsRecordOnly
```

The tests are unit tests against an in-memory SQLite database, isolated per
test. They need cgo but neither a Discord token nor network access.

New behaviour comes with tests, and a fix comes with a regression test.

## Pre-commit hooks

Hooks must pass before a commit is pushed. `--no-verify` and `SKIP=` are not the
fix: correct the code, or change the configuration in the same pull request and
say why.

```bash
pre-commit run --all-files
```

```bash
pre-commit run gofmt --all-files
```

| Hook | Checks | Fix |
| --- | --- | --- |
| `trailing-whitespace`, `end-of-file-fixer` | whitespace | re-run, fixes itself |
| `check-yaml`, `yamllint` | YAML validity and style, see [`.yamllint.yaml`](.yamllint.yaml) | edit the file |
| `check-added-large-files`, `check-merge-conflict` | accidental content | remove it |
| `detect-private-key` | committed private keys | remove it and rotate the key |
| `actionlint` | GitHub Actions workflows | edit the workflow |
| `gofmt` | Go formatting | re-run, fixes itself |
| `go-vet` | suspicious Go constructs | edit the code |
| `no-co-authors` | `Co-Authored-By` and `Generated with` lines in commit messages | rewrite the message |

The hooks run in CI too, except `no-co-authors`: `pre-commit run` skips
`commit-msg` hooks.

## Continuous integration

| Workflow | Triggers on | What it does | Reproduce locally |
| --- | --- | --- | --- |
| [`main-mr.yml`](.github/workflows/main-mr.yml) | pull requests to `main` | build, vet, tests with `-race`, `golangci-lint`, pre-commit | the commands above, and `golangci-lint run ./...` |
| [`main.yml`](.github/workflows/main.yml) | pushes to `main` | the same build, vet and tests, then pushes `ghcr.io/haibread/godisco:latest` | `docker build -t godisco:dev .` |
| [`codeql-analysis.yml`](.github/workflows/codeql-analysis.yml) | pushes to `main`, pull requests, weekly | CodeQL analysis of the Go code | none |

Pushing the image needs the `GHCR_PAT` secret, a token allowed to write to the
GitHub container registry.

## Submitting a change

- Commit subjects are short and imperative, see `git log --oneline`.
- Update the documentation in the same pull request as the code it describes.
- Dependency updates come from [Renovate](https://docs.renovatebot.com/),
  configured in [`.github/renovate.json`](.github/renovate.json).
