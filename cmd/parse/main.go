// Command parse reads LaTeX from stdin (line-by-line) and writes one JSON
// AST per line to stdout, matching the upstream `parse` CLI's output format
// so the two can be byte-diffed for parity testing.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/luo-studio/go-tex/tex/parser"
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "parse:", err)
		os.Exit(1)
	}
}

type errEnvelope struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Input   string `json:"input"`
}

func run(in io.Reader, out io.Writer) error {
	w := bufio.NewWriter(out)
	defer w.Flush()
	s := bufio.NewScanner(in)
	s.Buffer(make([]byte, 1<<16), 1<<20)
	for s.Scan() {
		expr := strings.TrimSpace(s.Text())
		if expr == "" {
			continue
		}
		body, err := parser.Parse(expr)
		if err != nil {
			env, _ := json.Marshal(errEnvelope{Error: true, Message: err.Error(), Input: expr})
			w.Write(env)
			w.WriteByte('\n')
			continue
		}
		js, err := json.Marshal(body)
		if err != nil {
			env, _ := json.Marshal(errEnvelope{Error: true, Message: err.Error(), Input: expr})
			w.Write(env)
			w.WriteByte('\n')
			continue
		}
		w.Write(js)
		w.WriteByte('\n')
	}
	return s.Err()
}
