# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security vulnerability in techtell, please follow these steps:

### 1. Do NOT Create a Public Issue

Please do not create a public GitHub issue for security vulnerabilities. This helps protect users until a fix is available.

### 2. Report Privately

Send your security report via email to:
- **Email**: [security@entro314-labs.com]
- **Subject**: `[SECURITY] techtell Vulnerability Report`

### 3. Include Detailed Information

Please include the following information in your report:

- **Description**: A detailed description of the vulnerability
- **Impact**: What could an attacker accomplish by exploiting this vulnerability?
- **Reproduction**: Step-by-step instructions to reproduce the vulnerability
- **Environment**: Operating system, techtell version, Go version
- **Supporting Material**: Screenshots, logs, or proof-of-concept code (if applicable)

### 4. Response Timeline

- **Acknowledgment**: We will acknowledge your report within 48 hours
- **Initial Assessment**: We will provide an initial assessment within 5 business days
- **Status Updates**: We will provide regular updates throughout the investigation
- **Resolution**: We aim to resolve critical vulnerabilities within 30 days

### 5. Coordinated Disclosure

1. We will work with you to understand and validate the vulnerability
2. We will develop and test a fix
3. We will coordinate the public disclosure timeline with you
4. We will credit you in our security advisory (if desired)

## Security Features

### Supply Chain Security
- Release checksums are signed with GPG
- Published container images are signed with keyless Cosign
- Published with SHA256 checksums for verification
- SBOM (Software Bill of Materials) attached to every release
- Dependencies monitored via `govulncheck` in CI

### Verification

```bash
# Verify checksums
sha256sum -c checksums.txt

# Import the published key only after comparing its fingerprint with a trusted source
gpg --import .release-public-key.asc
gpg --verify checksums.txt.sig checksums.txt

# Verify a container image signed by the release workflow
cosign verify ghcr.io/entro314-labs/techtell:<version> \
  --certificate-identity-regexp '^https://github.com/entro314-labs/techtell/.github/workflows/release\.yml@refs/(tags/v[0-9].*|heads/main)$' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

## Dependencies

We actively monitor dependencies for security vulnerabilities:
- Automated `govulncheck` scanning in CI
- Regular dependency updates
- Prompt response to vulnerability disclosures

## Contact

- **Security Team**: security@entro314-labs.com
- **GitHub Issues**: For non-security bugs and features only
