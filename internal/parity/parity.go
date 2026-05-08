// Package parity holds helpers for the parity test harness that diffs
// go-tex pipeline outputs against the upstream RaTeX binaries. Tests under
// this package skip cleanly when the upstream binaries are unavailable.
package parity

import (
	"os"
	"path/filepath"
)

// FindUpstreamBin searches for an upstream RaTeX binary by name. It checks,
// in order:
//
//  1. The environment variable named upper(name) (e.g. RATEX_LEX for "lex").
//  2. $RATEX_TARGET_DIR/release/<name>.
//
// It returns the resolved absolute path or an empty string if nothing was
// found. PATH is intentionally not consulted: names like "lex" and "parse"
// collide with unrelated Unix utilities (flex's lex, etc.) and would cause
// parity tests to feed LaTeX into the wrong program rather than skipping.
func FindUpstreamBin(name string) string {
	envName := envVarFor(name)
	if p := os.Getenv(envName); p != "" {
		if abs, err := filepath.Abs(p); err == nil {
			if fi, err := os.Stat(abs); err == nil && !fi.IsDir() {
				return abs
			}
		}
	}
	if dir := os.Getenv("RATEX_TARGET_DIR"); dir != "" {
		candidate := filepath.Join(dir, "release", name)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
	}
	return ""
}

func envVarFor(name string) string {
	out := make([]byte, 0, len(name)+6)
	out = append(out, "RATEX_"...)
	for _, b := range []byte(name) {
		switch {
		case b >= 'a' && b <= 'z':
			out = append(out, b-('a'-'A'))
		case b == '-':
			out = append(out, '_')
		default:
			out = append(out, b)
		}
	}
	return string(out)
}
