# Vulnerability Scanning Policy (Grype/SBOM)

## Pipeline behavior
- **CI (`.github/workflows/ci.yml`)**: report-only vulnerability visibility.
  - Generates SBOM + Grype JSON artifacts.
  - Prints severity summaries in logs.
  - Does **not** fail CI on findings.
- **Release (`.github/workflows/release-governance.yml`)**: strict gate.
  - Uses the same `.grype.yaml` baseline.
  - Fails on **High/Critical** (`--fail-on high`) unless ignored by policy.

## Baseline governance (`.grype.yaml`)
- Shared baseline for CI and release scans.
- Keep allowlist entries minimal and temporary.
- Each ignore entry must include:
  - `vulnerability` (CVE/advisory)
  - `reason` (why risk is temporarily accepted)
  - `until` (expiry date, `YYYY-MM-DD`)
  - `package.name` when practical

Expectation: prioritize remediation over allowlisting; all exceptions must expire.
