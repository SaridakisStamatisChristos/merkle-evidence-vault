package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

func b64u(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func makeJWK(pub *rsa.PublicKey) map[string]interface{} {
	n := pub.N.Bytes()
	e := big.NewInt(int64(pub.E)).Bytes()
	return map[string]interface{}{
		"kty": "RSA",
		"kid": "ci-test-key",
		"use": "sig",
		"alg": "RS256",
		"n":   b64u(n),
		"e":   b64u(e),
	}
}

func main() {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("failed to generate key: %v", err)
	}

	jwk := makeJWK(&priv.PublicKey)
	jwks := map[string]interface{}{"keys": []interface{}{jwk}}

	// create two tokens
	now := time.Now()
	makeToken := func(sub string, roles []string) string {
		claims := jwt.MapClaims{
			"sub":   sub,
			"roles": roles,
			"iat":   now.Unix(),
			"exp":   now.Add(time.Hour).Unix(),
			"iss":   "ci-jwks",
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tok.Header["kid"] = "ci-test-key"
		s, err := tok.SignedString(priv)
		if err != nil {
			log.Fatalf("failed to sign token: %v", err)
		}
		return s
	}

	ingester := makeToken("ci-ingester", []string{"ingester"})
	auditor := makeToken("ci-auditor", []string{"auditor"})

	// write env file for CI consumption
	envFile := []byte(fmt.Sprintf("E2E_INGESTER_TOKEN=%s\nE2E_AUDITOR_TOKEN=%s\nJWKS_URL=http://localhost:8000/jwks.json\n", ingester, auditor))
	if err := os.WriteFile("scripts/ci_jwks_env.txt", envFile, 0o600); err != nil {
		log.Printf("warning: failed to write env file: %v", err)
	}

	jwksBytes, _ := json.Marshal(jwks)

	// HTTP handlers
	http.HandleFunc("/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksBytes)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	// print tokens to stdout so CI step can capture
	fmt.Printf("E2E_INGESTER_TOKEN=%s\n", ingester)
	fmt.Printf("E2E_AUDITOR_TOKEN=%s\n", auditor)
	fmt.Printf("JWKS_URL=http://localhost:8000/jwks.json\n")

	log.Printf("serving jwks on :8000/jwks.json")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

// (no helper needed; using os.WriteFile inline)
