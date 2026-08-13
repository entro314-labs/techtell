# Release Signing 360° (Local Authority + Multi-CI)

This project signs release artifacts (checksums and archives) using GPG through GoReleaser.

## 1) Authoritative source of truth

Use your local machine (or hardware-backed local key setup) as the **authoritative source** for your release key.

- Generate and store the key locally.
- Keep a secure offline backup + revocation certificate.
- Export key material to CI secrets only (never commit keys).

### Suggested local key model

- One dedicated key for release signing (not personal email encryption key).
- Protect key with strong passphrase.
- Rotate periodically (e.g., yearly) and keep previous public keys for verification of old releases.

## 2) Provision CI secrets from local key

Export CI secret material from your local keyring:

```bash
bash ./scripts/release/export-gpg-key-for-ci.sh --fingerprint <YOUR_KEY_ID_OR_FINGERPRINT>
```

Then set in CI secrets:

- `GPG_PRIVATE_KEY` (preferred; ASCII-armored private key content)
  - or `GPG_PRIVATE_KEY_B64` (base64 fallback)
- `GPG_PASSWORD` (the GPG key passphrase)

## 3) CI setup pattern (portable)

In each CI pipeline before `goreleaser release`:

1. Normalize `GNUPGHOME` (`~/.gnupg` must be a directory)
2. Import key from secret env
3. Set trust/fingerprint non-interactively
4. Export `.release-public-key.asc` for publication with the release
5. Run GoReleaser signing

This repository uses:

```bash
bash ./scripts/release/setup-gpg-ci.sh
```

## 4) Reuse across multiple projects

Keep a reviewed copy of `scripts/release/setup-gpg-ci.sh` in each project so the workflow and signing contract change together. For non-GitHub CI providers, the same script works as long as the required environment variables are present.

Required env vars:

- `GPG_PRIVATE_KEY` (or `GPG_PRIVATE_KEY_B64`)
- `GPG_PASSWORD`

## 5) Verification for users

Each release publishes `checksums.txt`, `checksums.txt.sig`, and `.release-public-key.asc`. Users must compare the key fingerprint with a trusted, independently published fingerprint before importing it, then run:

```bash
gpg --import .release-public-key.asc
gpg --verify checksums.txt.sig checksums.txt
```

Container images are signed separately with keyless Cosign using the GitHub Actions OIDC identity. Verify that identity and issuer explicitly:

```bash
cosign verify ghcr.io/entro314-labs/techtell:<version> \
  --certificate-identity-regexp '^https://github.com/entro314-labs/techtell/.github/workflows/release\.yml@refs/(tags/v[0-9].*|heads/main)$' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

## SSH signing vs GPG signing

These are **separate** systems:

- **SSH commit/tag signing (GitHub UI “Verified”)**
  - Authenticates git objects (commits/tags)
  - Uses SSH keys
- **GPG artifact signing (GoReleaser `signs`)**
  - Authenticates release files/checksums
  - Uses GPG keys

They complement each other, but one does **not** replace the other.
