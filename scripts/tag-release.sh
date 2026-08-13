#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/tag-release.sh --bump <major|minor|patch> [options]
  scripts/tag-release.sh <major|minor|patch> [options]

Options:
  -b, --bump <type>      Version bump type: major, minor, patch
      --major            Bump major version
      --minor            Bump minor version
      --patch            Bump patch version
  -r, --remote <name>    Git remote to push tag to (default: origin)
  -p, --prefix <prefix>  Tag prefix (default: v)
  -m, --message <text>   Annotated tag message (default: "release: <tag>")
      --dry-run          Print actions without creating/pushing tag
      --allow-dirty      Allow uncommitted changes in working tree
  -h, --help             Show this help

Examples:
  scripts/tag-release.sh --bump patch
  scripts/tag-release.sh minor --dry-run
  scripts/tag-release.sh --major --remote upstream
EOF
}

fail() {
  echo "Error: $*" >&2
  exit 1
}

bump_type=""
remote="origin"
prefix="v"
message=""
dry_run="false"
allow_dirty="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    major|minor|patch)
      [[ -n "$bump_type" ]] && fail "Bump type already set to '$bump_type'"
      bump_type="$1"
      ;;
    -b|--bump)
      [[ $# -lt 2 ]] && fail "Missing value for $1"
      [[ -n "$bump_type" ]] && fail "Bump type already set to '$bump_type'"
      bump_type="$2"
      shift
      ;;
    --major)
      [[ -n "$bump_type" ]] && fail "Bump type already set to '$bump_type'"
      bump_type="major"
      ;;
    --minor)
      [[ -n "$bump_type" ]] && fail "Bump type already set to '$bump_type'"
      bump_type="minor"
      ;;
    --patch)
      [[ -n "$bump_type" ]] && fail "Bump type already set to '$bump_type'"
      bump_type="patch"
      ;;
    -r|--remote)
      [[ $# -lt 2 ]] && fail "Missing value for $1"
      remote="$2"
      shift
      ;;
    -p|--prefix)
      [[ $# -lt 2 ]] && fail "Missing value for $1"
      prefix="$2"
      shift
      ;;
    -m|--message)
      [[ $# -lt 2 ]] && fail "Missing value for $1"
      message="$2"
      shift
      ;;
    --dry-run)
      dry_run="true"
      ;;
    --allow-dirty)
      allow_dirty="true"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "Unknown argument '$1'"
      ;;
  esac
  shift
done

case "$bump_type" in
  major|minor|patch) ;;
  *) fail "You must provide --bump <major|minor|patch> (or positional major/minor/patch)" ;;
esac

git rev-parse --is-inside-work-tree >/dev/null 2>&1 || fail "Not inside a git repository"

if [[ "$allow_dirty" != "true" ]]; then
  if [[ -n "$(git status --porcelain)" ]]; then
    fail "Working tree is dirty. Commit/stash changes or pass --allow-dirty"
  fi
fi

latest_tag="$(git tag --list "${prefix}[0-9]*.[0-9]*.[0-9]*" --sort=-v:refname | head -n 1)"
if [[ -z "$latest_tag" ]]; then
  latest_tag="${prefix}0.0.0"
fi

base_version="${latest_tag#"${prefix}"}"
if [[ ! "$base_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  fail "Latest tag '$latest_tag' is not a valid semantic version"
fi

IFS='.' read -r major minor patch <<< "$base_version"

case "$bump_type" in
  major)
    major=$((major + 1))
    minor=0
    patch=0
    ;;
  minor)
    minor=$((minor + 1))
    patch=0
    ;;
  patch)
    patch=$((patch + 1))
    ;;
esac

new_tag="${prefix}${major}.${minor}.${patch}"

if git tag --list "$new_tag" | grep -qx "$new_tag"; then
  fail "Tag '$new_tag' already exists"
fi

existing_head_tag="$(git tag --points-at HEAD --list "${prefix}[0-9]*.[0-9]*.[0-9]*" | head -n 1)"
if [[ -n "$existing_head_tag" ]]; then
  echo "Note: HEAD already has semver tag '${existing_head_tag}'. Creating additional tag '${new_tag}'."
fi

if [[ -z "$message" ]]; then
  message="release: ${new_tag}"
fi

echo "Latest semver tag: ${latest_tag}"
echo "Bump type:         ${bump_type}"
echo "New tag:           ${new_tag}"
echo "Remote:            ${remote}"

if [[ "$dry_run" == "true" ]]; then
  echo "Dry run enabled. No changes made."
  exit 0
fi

git tag -a "$new_tag" -m "$message"
if ! git push "$remote" "$new_tag"; then
  git tag -d "$new_tag" >/dev/null
  fail "Push failed; removed local tag '$new_tag' so the release can be retried safely"
fi

echo "Done: created and pushed tag '${new_tag}'"
