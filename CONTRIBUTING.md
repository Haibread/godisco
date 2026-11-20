# Contributing

Open an issue or a pull request on
[GitHub](https://github.com/Haibread/godisco). A change lands once the checks
below are green.

## Development setup

- [Go](https://go.dev/doc/install) 1.26 or newer
- [pre-commit](https://pre-commit.com/#install), with
  [hadolint](https://github.com/hadolint/hadolint#install) and
  [Trivy](https://trivy.dev/latest/getting-started/installation/) on `PATH` for
  the Dockerfile hooks
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
| `shellcheck` | shell scripts, such as `scripts/release.sh` | edit the script |
| `hadolint`, `trivy-dockerfile` | `Dockerfile` lint and misconfigurations, with `hadolint` and `trivy` from `PATH` | edit the `Dockerfile` |
| `gofmt` | Go formatting | re-run, fixes itself |
| `go-vet` | suspicious Go constructs | edit the code |
| `no-co-authors` | `Co-Authored-By` and `Generated with` lines in commit messages | rewrite the message |

The hooks run in CI too, except `no-co-authors`: `pre-commit run` skips
`commit-msg` hooks.

## Continuous integration

| Workflow | Purpose | Reproduce locally |
| --- | --- | --- |
| [`quality.yaml`](.github/workflows/quality.yaml) | quality gate: pre-commit, build and tests with `-race`, `golangci-lint` | `pre-commit run --all-files`, `go test -race ./...`, `golangci-lint run ./...` |
| [`build.yaml`](.github/workflows/build.yaml) | image build and Trivy scan; pushes only when called by `release.yaml` or run by hand with `push` on `main` | `docker build -t godisco:dev .`, then `trivy image --severity HIGH,CRITICAL --ignore-unfixed godisco:dev` |
| [`security.yaml`](.github/workflows/security.yaml) | Trivy scan of the dependencies, CodeQL, and a daily scan of the latest released image | `trivy fs --severity HIGH,CRITICAL --ignore-unfixed .` |
| [`release.yaml`](.github/workflows/release.yaml) | on a `v*` tag: version check, image publish, GitHub release | none |
| [`scorecard.yaml`](.github/workflows/scorecard.yaml) | OpenSSF Scorecard grading of the repository | none |

`build.yaml` only runs when Go code, `go.mod`, `go.sum`, the `Dockerfile` or
`.dockerignore` change, so only `quality` and `security`, which run on every
pull request, can be required checks. `security` also runs daily, and goes
red without any change when a new fixable `HIGH` or `CRITICAL` CVE appears.

The workflows use no secret: pushing to GHCR uses the workflow's
`GITHUB_TOKEN`, and signing the attestations uses its OIDC identity. Code
scanning uploads are skipped on pull requests from forks, whose token is
read-only.

## Submitting a change

- Commit subjects are short and imperative, see `git log --oneline`.
- Update the documentation in the same pull request as the code it describes.
- Dependency updates come from [Renovate](https://docs.renovatebot.com/),
  configured in [`.github/renovate.json`](.github/renovate.json).
- Label the pull request with one category from
  [`.github/release.yaml`](.github/release.yaml), plus `breaking` when it
  breaks something: the release notes are generated from these labels.

## Releasing

Releases are cut with [`scripts/release.sh`](scripts/release.sh), the only
supported way: it writes the version into
[`app/godisco/version.go`](app/godisco/version.go), commits on `main`, tags
`v<version>`, then asks before pushing.

```bash
scripts/release.sh 1.2.0
```

```bash
scripts/release.sh --dry-run 1.2.0
```

The pushed tag runs [`release.yaml`](.github/workflows/release.yaml), which
refuses a tag that disagrees with `version.go`, publishes
`ghcr.io/haibread/godisco:<version>`, moves `:latest`, and creates the GitHub
release with generated notes and the SBOM. A pre-release such as
`1.3.0-rc.1` publishes `ghcr.io/haibread/godisco:1.3.0-rc.1` only: no
`:latest`, no GitHub release.
