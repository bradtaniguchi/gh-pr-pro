# Security Policy

## Supported Versions

We provide security updates and bug fixes for the latest release of `gh-pr-pro`. Users are encouraged to always run the most recent version.

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < Latest| :x:                |

---

## Reporting a Vulnerability

The maintainers of `gh-pr-pro` take security issues seriously. If you discover or suspect a security vulnerability in this project, please do **not** open a public issue or discuss it in public channels.

Please report all security vulnerabilities directly through **GitHub Private Vulnerability Reporting**:

1. Navigate to the repository's [**Security**](https://github.com/bradtaniguchi/gh-pr-pro/security) tab.
2. Click on [**Advisories**](https://github.com/bradtaniguchi/gh-pr-pro/security/advisories) and select **Report a vulnerability**.
3. Complete the advisory form with relevant technical details, steps to reproduce, and impact assessment.

GitHub Private Vulnerability Reporting is the fastest and only supported channel for disclosing security vulnerabilities in this project. It ensures secure, direct coordination with repository maintainers without exposing vulnerability details prematurely.

### What to Include in Your Report

To help us triage and resolve the issue quickly, please provide:
- A clear description of the vulnerability and its potential impact.
- Step-by-step instructions to reproduce the issue (including sample CLI commands, test repository configurations, or proof-of-concept scripts).
- The version of `gh-pr-pro`, `gh` CLI, and operating system used.
- Any known mitigations or workarounds.

### Out of Scope

The following scenarios are generally considered out of scope:
- Vulnerabilities requiring physical or root/administrative access to the local machine.
- Vulnerabilities or service disruptions in GitHub's REST/GraphQL APIs or upstream GitHub CLI (`gh`).
- Theoretical attacks without a working proof-of-concept (PoC) demonstrating practical impact.
- Vulnerabilities in third-party dependencies that have no reachable exploit path within `gh-pr-pro`.

---

## Disclosure and Response Timeline

We adhere to standard coordinated vulnerability disclosure (CVD) practices:

- **Initial Response**: We aim to acknowledge receipt of your report within **2–3 business days** via GitHub.
- **Assessment & Triage**: We will assess severity, confirm reproducibility, and establish an action plan within **5–7 business days**.
- **Remediation**:
  - **Critical / High Severity**: Target patch and release within **14–30 days**.
  - **Medium / Low Severity**: Target patch within **30–90 days** (or the next minor release).
- **Public Disclosure**: We coordinate public disclosure with the reporter via GitHub Security Advisories, observing a **90-day maximum disclosure window** unless mutually agreed otherwise.

---

## Security Considerations for `gh-pr-pro`

- **Authentication & Tokens**: `gh-pr-pro` utilizes the GitHub CLI (`gh`) authentication session and does not log, persist, or expose authentication tokens. Ensure your local `gh` token grants only the necessary scopes (`repo`, `read:org`).
- **Disk Cache**: Metric calculations are cached locally under `~/.cache/gh-pr-pro/` with standard user file permissions. Avoid running on shared multi-user systems without appropriate file system isolation if analyzing private repository metrics.
