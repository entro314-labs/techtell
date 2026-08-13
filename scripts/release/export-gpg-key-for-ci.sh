#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Export a local GPG release key for CI secret provisioning.

Usage:
  scripts/release/export-gpg-key-for-ci.sh --fingerprint <key-id-or-fpr> [--out-dir <dir>]

Options:
  -f, --fingerprint <id>  Secret key id/fingerprint to export
  -o, --out-dir <dir>     Output directory (default: .release-secrets)
  -h, --help              Show help

Outputs:
  <out-dir>/GPG_PRIVATE_KEY.asc
  <out-dir>/GPG_PRIVATE_KEY_B64.txt
EOF
}

fail() {
  echo "Error: $*" >&2
  exit 1
}

fingerprint=""
out_dir=".release-secrets"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -f|--fingerprint)
      [[ $# -lt 2 ]] && fail "Missing value for $1"
      fingerprint="$2"
      shift
      ;;
    -o|--out-dir)
      [[ $# -lt 2 ]] && fail "Missing value for $1"
      out_dir="$2"
      shift
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

[[ -n "$fingerprint" ]] || fail "--fingerprint is required"

mkdir -p "$out_dir"

asc_file="$out_dir/GPG_PRIVATE_KEY.asc"
b64_file="$out_dir/GPG_PRIVATE_KEY_B64.txt"

gpg --batch --armor --export-secret-keys "$fingerprint" > "$asc_file"
base64 < "$asc_file" | tr -d '\n' > "$b64_file"

echo "Export complete:"
echo "  - $asc_file"
echo "  - $b64_file"
echo
echo "Set CI secret 'GPG_PRIVATE_KEY' to the .asc content (preferred)."
echo "Or set 'GPG_PRIVATE_KEY_B64' to the .txt content."
echo "Set CI secret 'GPG_PASSWORD' to the key passphrase used for signing."
