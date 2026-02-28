package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
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
	noRoles := makeToken("ci-no-roles", nil)

	jwksBytes, _ := json.Marshal(jwks)

	// Persist JWKS to disk for CI debugging and comparative checks
	if err := os.WriteFile("scripts/jwks.json", jwksBytes, 0o600); err != nil {
		log.Printf("warning: failed to write jwks.json: %v", err)
	}

	// HTTP handlers
	http.HandleFunc("/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksBytes)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	// determine port to listen on
	port := os.Getenv("CI_JWKS_PORT")
	if port == "" {
		port = "0" // let OS choose an available port
	}

	// Ensure scripts directory exists so we can write the env file.
	if err := os.MkdirAll("scripts", 0755); err != nil {
		log.Printf("warning: failed to ensure scripts dir: %v", err)
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Printf("warning: failed to listen on port %s: %v; trying random port", port, err)
		ln, err = net.Listen("tcp", ":0")
		if err != nil {
			log.Fatalf("failed to listen on fallback port: %v", err)
		}
	}
	_, actualPort, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		log.Fatalf("failed to parse listener address: %v", err)
	}

	// write env file for CI consumption with correct port
	envFile := []byte(fmt.Sprintf(
		"E2E_INGESTER_TOKEN=%s\nE2E_AUDITOR_TOKEN=%s\nE2E_NO_ROLES_TOKEN=%s\nJWKS_URL=http://localhost:%s/jwks.json\n",
		ingester, auditor, noRoles, actualPort))
	if err := os.WriteFile("scripts/ci_jwks_env.txt", envFile, 0o600); err != nil {
		log.Printf("warning: failed to write env file: %v", err)
	}

	// print tokens to stdout so CI step can capture
	fmt.Printf("E2E_INGESTER_TOKEN=%s\n", ingester)
	fmt.Printf("E2E_AUDITOR_TOKEN=%s\n", auditor)
	fmt.Printf("E2E_NO_ROLES_TOKEN=%s\n", noRoles)
	fmt.Printf("JWKS_URL=http://localhost:%s/jwks.json\n", actualPort)

	log.Printf("serving jwks on :%s/jwks.json", actualPort)
	log.Fatal(http.Serve(ln, nil))
}
