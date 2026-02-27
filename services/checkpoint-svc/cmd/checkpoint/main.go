package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/SaridakisStamatisChristos/checkpoint-svc/metrics"
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

	var signerObj signer.Signer
	var signerErr error
	kmsProvider := os.Getenv("KMS_PROVIDER")
	kmsKeyID := os.Getenv("KMS_KEY_ID")
	if kmsProvider != "" && kmsKeyID != "" {
		s, err := signer.NewKMSSigner(kmsProvider, kmsKeyID)
		if err != nil {
			signerErr = err
		} else {
			signerObj = s
		}
	} else if keyB64 := os.Getenv("CHECKPOINT_PRIVATE_KEY_B64"); keyB64 != "" {
		s, err := signer.NewLocalSignerFromBase64WithRef(keyB64, "local:env:CHECKPOINT_PRIVATE_KEY_B64")
		if err != nil {
			signerErr = err
		} else {
			signerObj = s
		}
	} else {
		b, err := os.ReadFile(*keyPath)
		if err != nil {
			signerErr = err
		} else {
			s, err := signer.NewLocalSignerFromBase64WithRef(string(b), "local:file:"+*keyPath)
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

	http.Handle("/metrics", metrics.Handler())

	if signerObj != nil {
		http.HandleFunc("/sign", func(w http.ResponseWriter, r *http.Request) {
			metrics.IncSignRequests()
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			b, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			sig, err := signerObj.Sign(b)
			if err != nil {
				log.Printf("sign error key_ref=%s err=%v", signerObj.KeyRef(), err)
				metrics.IncSignFailures()
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			log.Printf("audit event=checkpoint_sign key_ref=%s payload_bytes=%d", signerObj.KeyRef(), len(b))
			enc := base64.StdEncoding.EncodeToString(sig)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Checkpoint-Key-Ref", signerObj.KeyRef())
			json.NewEncoder(w).Encode(map[string]string{"signature": enc, "key_ref": signerObj.KeyRef()})
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
			log.Println("emit checkpoint (stub)")
		case <-ctx.Done():
			log.Println("shutting down")
			return
		}
	}
}
