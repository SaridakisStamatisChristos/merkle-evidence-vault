# Contributing

## Pull request process hygiene

All pull requests should follow these conventions:

1. Fill out the PR template sections using markdown headings exactly as `### Summary`, `### Checklist`, and `### Test Notes`.
2. Keep the summary in a release-note style: what changed and why.
3. Apply at least one scope label (`backend`, `frontend`, `ci`, `infra`, `security`) and one change-type label where relevant.
4. Request review from CODEOWNERS and any impacted domain owner.
5. Include commands used for validation in the PR body.

The `PR Hygiene` workflow enforces template sections and auto-applies path-based labels via `.github/labeler.yml`.
