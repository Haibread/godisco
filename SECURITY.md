# Security policy

## Reporting a vulnerability

Report vulnerabilities privately through
[GitHub's private vulnerability reporting](https://github.com/Haibread/godisco/security/advisories/new).
Only the maintainers and you see the report.

Do not open a public issue, pull request or discussion: a vulnerability
described in public is disclosed before it is fixed.

A useful report holds:

- the affected version or commit;
- the steps to reproduce;
- the impact as you see it.

## Supported versions

The default branch is the only supported version. Fixes land there and ship in
the `ghcr.io/haibread/godisco:latest` image.

## Scope

In scope: godisco's own code and the container image it publishes, including
how it handles the Discord bot token in `config/config.yaml`, Discord events
and slash command input, and the SQLite database.

Out of scope:

- a vulnerability in a dependency that is already public upstream: report it
  there, Renovate brings the fix in;
- a finding that needs an already-compromised host, Discord account or bot
  token;
- scanner output with no path to exploitation.

## What happens next

Reports are handled on a best-effort basis.

1. The report is acknowledged.
2. The maintainers assess it.
3. The fix is prepared in the advisory's private fork.
4. The fix is released.
5. The advisory is published, with a CVE requested through GitHub when the
   impact warrants one, and you are credited unless you prefer otherwise.
