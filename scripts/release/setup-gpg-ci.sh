#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Prepare GnuPG in CI and import signing key from environment secrets.

Environment variables:
  GPG_PRIVATE_KEY       ASCII-armored private key (preferred)
  GPG_PRIVATE_KEY_B64   Base64-encoded ASCII-armored private key (fallback)
  GNUPGHOME             Optional custom gnupg home (default: $HOME/.gnupg)

Options:
  --allow-missing-key   Do not fail if key env vars are missing
  -h, --help            Show help
EOF
}

fail() {
  echo "Error: $*" >&2
  exit 1
}

allow_missing_key="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --allow-missing-key)
      allow_missing_key="true"
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

GNUPGHOME="${GNUPGHOME:-$HOME/.gnupg}"

if [[ -e "$GNUPGHOME" && ! -d "$GNUPGHOME" ]]; then
  rm -f "$GNUPGHOME"
fi

mkdir -p "$GNUPGHOME"
chmod 700 "$GNUPGHOME"

# Persist for subsequent CI steps where supported.
if [[ -n "${GITHUB_ENV:-}" ]]; then
  echo "GNUPGHOME=$GNUPGHOME" >> "$GITHUB_ENV"
fi

key_material=""
if [[ -n "${GPG_PRIVATE_KEY:-}" ]]; then
  key_material="${GPG_PRIVATE_KEY}"
elif [[ -n "${GPG_PRIVATE_KEY_B64:-}" ]]; then
  key_material="$(printf '%s' "${GPG_PRIVATE_KEY_B64}" | base64 --decode)"
fi

if [[ -z "$key_material" ]]; then
  if [[ "$allow_missing_key" == "true" ]]; then
    echo "No GPG key provided via GPG_PRIVATE_KEY/GPG_PRIVATE_KEY_B64; skipping import."
    exit 0
  fi
  fail "Missing GPG key. Set GPG_PRIVATE_KEY (preferred) or GPG_PRIVATE_KEY_B64 in CI secrets."
fi

tmp_key_file="$(mktemp)"
trap 'rm -f "$tmp_key_file"' EXIT
printf '%s\n' "$key_material" > "$tmp_key_file"

gpg --batch --import "$tmp_key_file"

fingerprint="$(gpg --batch --list-secret-keys --with-colons | awk -F: '/^fpr:/ {print $10; exit}')"
if [[ -n "$fingerprint" ]]; then
  # Ultimate trust to avoid non-interactive trust prompts in CI.
  printf '%s:6:\n' "$fingerprint" | gpg --batch --import-ownertrust >/dev/null 2>&1 || true
  if [[ -n "${GITHUB_ENV:-}" ]]; then
    echo "GPG_FINGERPRINT=$fingerprint" >> "$GITHUB_ENV"
  fi

  public_key_file="${GITHUB_WORKSPACE:-$PWD}/.release-public-key.asc"
  gpg --batch --armor --export "$fingerprint" > "$public_key_file"
  [[ -s "$public_key_file" ]] || fail "Failed to export the release public key"
fi

echo "GPG setup complete${fingerprint:+ (fingerprint: $fingerprint)}"
