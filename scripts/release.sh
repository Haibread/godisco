#!/usr/bin/env bash
# Cut a godisco release: write the version, commit, tag, then ask before
# pushing. Pushing the tag is what makes CI publish the image.
set -euo pipefail

readonly version_file="app/godisco/version.go"
readonly default_branch="main"

usage() {
	cat <<'EOF'
Usage: scripts/release.sh [--dry-run] <version>

Write <version> into app/godisco/version.go, commit it on main, tag it
v<version>, then ask before pushing both to origin. Pushing the tag runs
.github/workflows/release.yaml, which publishes the image.

<version> is semver without the leading v. A pre-release (1.2.0-rc.1)
publishes the image only, with no GitHub release and no move of :latest.

Options:
  --dry-run   print what would change, change nothing
  -h, --help  show this help

Examples:
  scripts/release.sh 1.2.0
  scripts/release.sh 1.3.0-rc.1
  scripts/release.sh --dry-run 1.2.1
EOF
}

die() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}

dry_run=false
target=""
while [ $# -gt 0 ]; do
	case "$1" in
	-h | --help)
		usage
		exit 0
		;;
	--dry-run)
		dry_run=true
		;;
	-*)
		die "unknown option $1, see --help"
		;;
	*)
		[ -z "$target" ] || die "one version only, see --help"
		target="$1"
		;;
	esac
	shift
done

[ -n "$target" ] || die "missing version, see --help"

semver='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$'
[[ "$target" =~ $semver ]] || die "'$target' is not a semver version such as 1.2.0 (no leading v)"

cd "$(git rev-parse --show-toplevel)"

tag="v${target}"
current="$(sed -n 's/^const version = "\(.*\)"$/\1/p' "$version_file")"
[ -n "$current" ] || die "no 'const version = \"...\"' line in ${version_file}"

if [ "$dry_run" = true ]; then
	printf 'version: %s -> %s\n' "$current" "$target"
	printf 'commit:  chore: release %s\n' "$tag"
	printf 'tag:     %s\n' "$tag"
	exit 0
fi

[ -z "$(git status --porcelain)" ] || die "the working tree is not clean, commit or stash first"

branch="$(git symbolic-ref --short HEAD)"
[ "$branch" = "$default_branch" ] || die "on '${branch}', releases are cut from ${default_branch}"

git fetch --quiet --tags origin "$default_branch"
behind="$(git rev-list --count "HEAD..origin/${default_branch}")"
[ "$behind" -eq 0 ] || die "${default_branch} is ${behind} commit(s) behind origin, pull first"

if git rev-parse --quiet --verify "refs/tags/${tag}" >/dev/null; then
	die "tag ${tag} already exists"
fi
[ "$current" != "$target" ] || die "${version_file} already says ${target}"

sed -i.bak "s/^const version = \".*\"$/const version = \"${target}\"/" "$version_file"
rm -f "${version_file}.bak"

git add "$version_file"
git commit --quiet --message "chore: release ${tag}"
git tag --annotate "$tag" --message "$tag"
printf 'committed and tagged %s\n' "$tag"

push_args=(--atomic origin "$default_branch" "$tag")
if [ ! -t 0 ]; then
	printf 'not a terminal, not pushing. To publish:\n  git push %s\n' "${push_args[*]}"
	exit 0
fi

printf 'push %s and %s to origin? this publishes the release. [y/N] ' "$default_branch" "$tag"
read -r reply
case "$reply" in
y | Y | yes | YES)
	git push "${push_args[@]}"
	;;
*)
	printf 'not pushed. To publish later:\n  git push %s\n' "${push_args[*]}"
	;;
esac
