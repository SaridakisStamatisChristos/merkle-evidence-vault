# Auth Policy Matrix

| AUTH_POLICY | ENV/DEPLOYMENT expectation | Required env vars | Startup behavior |
|---|---|---|---|
| `dev` | Allowed only in `ENV=dev`, or with `ALLOW_INSECURE_DEV=true` | none | Rejects startup in production profile. |
| `jwks_strict` | Any non-dev profile, including production | `JWKS_URL`, `JWT_ISSUER`, `JWT_AUDIENCE` | Fails fast if any required setting is missing. |
| `jwks_rbac` | Preferred for production role/claim enforcement | `JWKS_URL`, `JWT_ISSUER`, `JWT_AUDIENCE` | Fails fast if any required setting is missing. |

## Production guardrails

- `AUTH_POLICY=dev` is blocked when `ENV=prod` or `DEPLOYMENT=prod`.
- JWT/JWKS modes are startup-gated on required issuer/audience/JWKS settings.
- Policy intent: no permissive auth mode in production boot path.
