package middleware

import "testing"

func TestValidateAuthStartupConfig_ProdRejectsDevPolicy(t *testing.T) {
	t.Setenv("ENV", "prod")
	t.Setenv("AUTH_POLICY", authPolicyDev)
	t.Setenv("ALLOW_INSECURE_DEV", "false")

	err := ValidateAuthStartupConfig()
	if err == nil {
		t.Fatalf("expected startup error for prod + dev auth policy")
	}
}

func TestValidateAuthStartupConfig_DevPolicyAllowedInDevEnv(t *testing.T) {
	t.Setenv("ENV", "dev")
	t.Setenv("AUTH_POLICY", authPolicyDev)
	t.Setenv("ALLOW_INSECURE_DEV", "false")

	if err := ValidateAuthStartupConfig(); err != nil {
		t.Fatalf("expected dev policy to be allowed in dev env, got %v", err)
	}
}

func TestValidateAuthStartupConfig_DevPolicyAllowedWithInsecureOverride(t *testing.T) {
	t.Setenv("ENV", "staging")
	t.Setenv("AUTH_POLICY", authPolicyDev)
	t.Setenv("ALLOW_INSECURE_DEV", "true")

	if err := ValidateAuthStartupConfig(); err != nil {
		t.Fatalf("expected dev policy with ALLOW_INSECURE_DEV=true, got %v", err)
	}
}

func TestValidateAuthStartupConfig_StrictPolicyMissingRequiredVars(t *testing.T) {
	t.Setenv("ENV", "prod")
	t.Setenv("AUTH_POLICY", authPolicyJWKSStrict)
	t.Setenv("JWKS_URL", "")
	t.Setenv("JWT_ISSUER", "")
	t.Setenv("JWT_AUDIENCE", "")

	err := ValidateAuthStartupConfig()
	if err == nil {
		t.Fatalf("expected startup error when strict policy env vars are missing")
	}
}

func TestValidateAuthStartupConfig_RBACPolicyMissingRequiredVars(t *testing.T) {
	t.Setenv("DEPLOYMENT", "prod")
	t.Setenv("AUTH_POLICY", authPolicyJWKSRBAC)
	t.Setenv("JWKS_URL", "https://jwks.example/.well-known/jwks.json")
	t.Setenv("JWT_ISSUER", "issuer-1")
	t.Setenv("JWT_AUDIENCE", "")

	err := ValidateAuthStartupConfig()
	if err == nil {
		t.Fatalf("expected startup error when rbac policy vars are incomplete")
	}
}

func TestValidateAuthStartupConfig_StrictPolicyWithRequiredVarsPasses(t *testing.T) {
	t.Setenv("ENV", "prod")
	t.Setenv("AUTH_POLICY", authPolicyJWKSStrict)
	t.Setenv("JWKS_URL", "https://jwks.example/.well-known/jwks.json")
	t.Setenv("JWT_ISSUER", "issuer-1")
	t.Setenv("JWT_AUDIENCE", "vault-api")

	if err := ValidateAuthStartupConfig(); err != nil {
		t.Fatalf("expected valid startup auth config, got %v", err)
	}
}
