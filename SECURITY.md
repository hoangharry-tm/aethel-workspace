# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.0.x   | ✅ Security patches delivered |
| < 1.0   | ❌ Not supported |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Use GitHub's private vulnerability reporting:

1. Go to the repository on GitHub
2. Click the **Security** tab
3. Click **Report a vulnerability**
4. Fill in the details and submit

Alternatively, email **tonminhhoang.work@gmail.com** with subject line `[SECURITY] Aethel Workspace — <short description>`. Encrypt sensitive details using the PGP key published in the repository if available.

Please include:
- Affected component (backend, frontend, auth, audit ledger, etc.)
- Steps to reproduce
- Potential impact and severity estimate (CVSS score if possible)
- Any suggested fix or mitigation

## Response SLA

| Severity | CVSS Score | Acknowledge | Triage | Patch Release |
|----------|-----------|-------------|--------|---------------|
| Critical / High | ≥ 7.0 | 48 hours | 7 days | 30 days |
| Medium / Low | < 7.0 | 48 hours | 14 days | 90 days |

We will keep reporters informed of progress at each stage.

## Scope

**In scope:**
- Authentication bypass (JWT forgery, session fixation, cookie theft)
- Authorization bypass (RBAC privilege escalation, `sys_admin` gating circumvention)
- SQL injection via API endpoints
- Cross-site scripting (XSS) in admin pages or document rendering
- Sensitive data exposure via API responses (passwords, JWT secrets, PII)
- Cryptographic weaknesses in the green note hash chain or audit ledger chain
- Remote code execution via file upload or template injection
- CSRF protection bypass

**Out of scope:**
- Rate limiting edge cases or denial-of-service without authentication
- UI cosmetics, broken links, or missing error messages
- Issues requiring physical access to the server
- Vulnerabilities in third-party dependencies — report those to the upstream project
- Issues only reproducible on unsupported versions (< 1.0)
- Social engineering attacks

## Credit

Security researchers who responsibly disclose vulnerabilities will be:
- Credited by name (or handle) in the relevant `CHANGELOG.md` entry
- Listed in the GitHub Security Advisory for the associated CVE

Reporters may request anonymity — just say so in your report and we will honour it.
