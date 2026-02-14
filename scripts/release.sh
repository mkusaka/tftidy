#!/usr/bin/env bash
set -euo pipefail

# Usage: ./scripts/release.sh patch|minor|major

arg="${1:-}"

if [ -z "${arg}" ]; then
  echo "Usage: $0 <patch|minor|major|vX.Y.Z>"
  echo ""
  echo "Recent tags:"
  git tag --sort=-version:refname | head -5 || echo "  (none)"
  exit 1
fi

if echo "${arg}" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
  # Explicit tag: ./scripts/release.sh v1.2.3
  tag="${arg}"
else
  # Bump: ./scripts/release.sh patch|minor|major
  case "${arg}" in
    patch|minor|major) ;;
    *) echo "Error: argument must be 'patch', 'minor', 'major', or 'vX.Y.Z', got '${arg}'"; exit 1 ;;
  esac

  latest=$(git tag --sort=-version:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1 || true)
  if [ -z "${latest}" ]; then
    latest="v0.0.0"
  fi

  version="${latest#v}"
  IFS='.' read -r major minor patch <<< "${version}"

  case "${arg}" in
    patch) patch=$((patch + 1)) ;;
    minor) minor=$((minor + 1)); patch=0 ;;
    major) major=$((major + 1)); minor=0; patch=0 ;;
  esac

  tag="v${major}.${minor}.${patch}"
fi

# Check for uncommitted changes
if [ -n "$(git status --porcelain)" ]; then
  echo "Error: working tree is not clean"
  git status --short
  exit 1
fi

# Check tag doesn't already exist
if git rev-parse "${tag}" >/dev/null 2>&1; then
  echo "Error: tag '${tag}' already exists"
  exit 1
fi

# Show what will be released
echo "${latest} -> ${tag} (${bump})"
echo "Commit: $(git log --oneline -1)"
echo "Branch: $(git branch --show-current)"
echo ""
read -rp "Proceed? [y/N] " confirm
if [ "${confirm}" != "y" ] && [ "${confirm}" != "Y" ]; then
  echo "Aborted."
  exit 0
fi

git tag "${tag}"
git push origin "${tag}"

echo ""
echo "Tag '${tag}' pushed. GitHub Actions will:"
echo "  1. Build binaries for 5 platforms via goreleaser"
echo "  2. Create GitHub Release with assets"
echo "  3. Update major version tag (v${major})"
echo ""
echo "Monitor: https://github.com/mkusaka/tftidy/actions"
