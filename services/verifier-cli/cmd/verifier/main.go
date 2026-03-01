package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type drillInput struct {
	ExpectedRoot string `json:"expected_root"`
	Before       struct {
		Checkpoint struct {
			TreeSize int64  `json:"tree_size"`
			RootHash string `json:"root_hash"`
		} `json:"checkpoint"`
		CheckpointVerify struct {
			Verified bool `json:"verified"`
		} `json:"checkpoint_verify"`
		Proofs []struct {
			ID       string `json:"id"`
			HTTPCode int    `json:"http_code"`
			Root     string `json:"root"`
		} `json:"proofs"`
	} `json:"before"`
	After struct {
		Checkpoint struct {
			TreeSize int64  `json:"tree_size"`
			RootHash string `json:"root_hash"`
		} `json:"checkpoint"`
		CheckpointVerify struct {
			Verified bool `json:"verified"`
		} `json:"checkpoint_verify"`
		Proofs []struct {
			ID       string `json:"id"`
			HTTPCode int    `json:"http_code"`
			Root     string `json:"root"`
		} `json:"proofs"`
	} `json:"after"`
}

type drillOutput struct {
	Pass                 bool     `json:"pass"`
	LatestRootMatches    bool     `json:"latest_root_matches_expected"`
	CheckpointSignatures bool     `json:"checkpoint_signatures_verify"`
	SampleProofsVerify   bool     `json:"sample_proofs_verify"`
	Failures             []string `json:"failures,omitempty"`
}

func main() {
	bundle := flag.String("bundle", "", "path to .evb bundle")
	pub := flag.String("public-key", "", "public key file (hex or PEM)")
	drillInputPath := flag.String("drill-input", "", "path to drill input JSON")
	outputPath := flag.String("output", "", "path to write verification JSON")
	flag.Parse()

	if *drillInputPath != "" {
		out, err := verifyDrill(*drillInputPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		if *outputPath != "" {
			if err := os.WriteFile(*outputPath, b, 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "failed writing output: %v\n", err)
				os.Exit(1)
			}
		}
		fmt.Println(string(b))
		if !out.Pass {
			os.Exit(1)
		}
		return
	}

	if *bundle == "" || *pub == "" {
		fmt.Fprintln(os.Stderr, "usage: verifier-cli --bundle <file> --public-key <file>")
		fmt.Fprintln(os.Stderr, "   or: verifier-cli --drill-input <json> [--output <json>]")
		os.Exit(2)
	}

	// TODO: open .evb, verify STH signature, verify inclusion proofs
	fmt.Printf("verifying %s with key %s (stub)\n", *bundle, *pub)
}

func verifyDrill(path string) (drillOutput, error) {
	var in drillInput
	b, err := os.ReadFile(path)
	if err != nil {
		return drillOutput{}, fmt.Errorf("read drill input: %w", err)
	}
	if err := json.Unmarshal(b, &in); err != nil {
		return drillOutput{}, fmt.Errorf("parse drill input: %w", err)
	}

	out := drillOutput{}
	out.LatestRootMatches = in.ExpectedRoot != "" && in.After.Checkpoint.RootHash == in.ExpectedRoot
	if !out.LatestRootMatches {
		out.Failures = append(out.Failures, "latest root hash mismatch after restore")
	}

	out.CheckpointSignatures = in.Before.CheckpointVerify.Verified && in.After.CheckpointVerify.Verified
	if !out.CheckpointSignatures {
		out.Failures = append(out.Failures, "checkpoint signature verification failed")
	}

	proofsOK := len(in.Before.Proofs) > 0 && len(in.After.Proofs) > 0
	if proofsOK {
		for _, p := range in.Before.Proofs {
			if p.HTTPCode != 200 || p.Root == "" {
				proofsOK = false
				break
			}
		}
		for _, p := range in.After.Proofs {
			if p.HTTPCode != 200 || p.Root == "" {
				proofsOK = false
				break
			}
		}
	}
	out.SampleProofsVerify = proofsOK
	if !out.SampleProofsVerify {
		out.Failures = append(out.Failures, "sample evidence proof verification failed")
	}

	out.Pass = out.LatestRootMatches && out.CheckpointSignatures && out.SampleProofsVerify
	return out, nil
}
