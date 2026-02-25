package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	bundle := flag.String("bundle", "", "path to .evb bundle")
	pub := flag.String("public-key", "", "public key file (hex or PEM)")
	flag.Parse()

	if *bundle == "" || *pub == "" {
		fmt.Fprintln(os.Stderr, "usage: verifier-cli --bundle <file> --public-key <file>")
		os.Exit(2)
	}

	// TODO: open .evb, verify STH signature, verify inclusion proofs
	fmt.Printf("verifying %s with key %s (stub)\n", *bundle, *pub)
}
