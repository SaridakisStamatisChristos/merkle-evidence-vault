package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"time"
	"github.com/SaridakisStamatisChristos/checkpoint-svc/signer"
)

func main() {
	interval := 300 * time.Second
	if v := os.Getenv("CHECKPOINT_INTERVAL_SECONDS"); v != "" {
		// ignore parse errors for stub; allow overriding via env
		interval = 60 * time.Second
	}

	keyPath := flag.String("key", "checkpoint_key.b64", "path to base64 private key")
	addr := flag.String("addr", ":8081", "http listen addr for signing endpoint")
	flag.Parse()
	// signer selection: local keyfile (default) or KMS-backed signing based on env
	var signerObj interface{}
	var signerErr error
	kmsProvider := os.Getenv("KMS_PROVIDER")
	kmsKeyID := os.Getenv("KMS_KEY_ID")
	if kmsProvider != "" && kmsKeyID != "" {
		// attempt to initialize KMS signer (stubbed)
		s, err := signer.NewKMSSigner(kmsProvider, kmsKeyID)
		if err != nil {
			signerErr = err
		} else {
			signerObj = s
		}
	} else {
		// fallback: local key file (base64)
		b, err := os.ReadFile(*keyPath)
		if err != nil {
			signerErr = err
		} else {
			s, err := signer.NewLocalSignerFromBase64(string(b))
			if err != nil {
				signerErr = err
			} else {
				signerObj = s
			}
		}
	}

	if signerObj == nil {
		log.Printf("warning: no signer available (%v); signing endpoint disabled", signerErr)
	}

	// signing handler
	if signerObj != nil {
		http.HandleFunc("/sign", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			b, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var sig []byte
			switch s := signerObj.(type) {
			case *signer.LocalSigner:
				sig, err = s.Sign(b)
			case *signer.KMSSigner:
				sig, err = s.Sign(b)
			default:
				err = nil
			}
			if err != nil {
				log.Printf("sign error: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			enc := base64.StdEncoding.EncodeToString(sig)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"signature": enc})
		})
		go func() {
			log.Printf("checkpoint-svc signing endpoint listening on %s", *addr)
			if err := http.ListenAndServe(*addr, nil); err != nil {
				log.Printf("signing server exited: %v", err)
			}
		}()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("checkpoint-svc starting, interval=%v", interval)
	for {
		select {
		case <-ticker.C:
			// TODO: call merkle-engine SignedTreeHead RPC and persist
			log.Println("emit checkpoint (stub)")
		case <-ctx.Done():
			log.Println("shutting down")
			return
		}
	}
}

func loadKey(path string) (ed25519.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// trim whitespace/newlines
	s := string(b)
	dec, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if len(dec) != ed25519.PrivateKeySize {
		return nil, err
	}
	return ed25519.PrivateKey(dec), nil
}
