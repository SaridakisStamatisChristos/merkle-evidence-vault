package main

import (
	"context"
	"log"
	"os"
	"time"
)

func main() {
	interval := 300 * time.Second
	if v := os.Getenv("CHECKPOINT_INTERVAL_SECONDS"); v != "" {
		// ignore parse errors for stub
		interval = 60 * time.Second
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
