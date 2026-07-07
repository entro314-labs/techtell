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
- `COSIGN_PWD` (the key passphrase)

## 3) CI setup pattern (portable)

In each CI pipeline before `goreleaser release`:

1. Normalize `GNUPGHOME` (`~/.gnupg` must be a directory)
2. Import key from secret env
3. Set trust/fingerprint non-interactively
4. Run GoReleaser signing

This repository uses:

```bash
bash ./scripts/release/setup-gpg-ci.sh
```

## 4) Reuse across multiple projects

To roll this out consistently:

- Publish a shared `ci-scripts` repository and tag versions (for example `v0.1.0`).
- In each project CI, fetch `release/setup-gpg-ci.sh` from the pinned tag.
- Keep `scripts/release/setup-gpg-ci.sh` in-project only as fallback/bootstrap.

For non-GitHub CI providers, the same script works as long as env vars are present.

Required env vars:

- `GPG_PRIVATE_KEY` (or `GPG_PRIVATE_KEY_B64`)
- `COSIGN_PWD`

GitHub Actions pattern:

```yaml
env:
  CI_SCRIPTS_REPO: idominikosgr/ci-scripts
  CI_SCRIPTS_VERSION: v0.1.0
```

## 5) Verification for users

Users should verify checksums/signatures and trust your public key fingerprint published in docs/release notes.

## SSH signing vs GPG signing

These are **separate** systems:

- **SSH commit/tag signing (GitHub UI “Verified”)**
  - Authenticates git objects (commits/tags)
  - Uses SSH keys
- **GPG artifact signing (GoReleaser `signs`)**
  - Authenticates release files/checksums
  - Uses GPG keys

They complement each other, but one does **not** replace the other.