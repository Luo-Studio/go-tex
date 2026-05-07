// Command lex reads LaTeX from stdin and writes one token text per line to
// stdout. Used by golden-output and lexer-compare tooling to diff against
// the upstream RaTeX/KaTeX lexer output.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/luo-studio/go-tex/tex/lexer"
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "lex:", err)
		os.Exit(1)
	}
}

func run(in io.Reader, out io.Writer) error {
	data, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	// Strip a single trailing newline so `echo -n 'x^2'` and `echo 'x^2'`
	// behave the same — matching the upstream Rust CLI.
	input := strings.TrimSuffix(string(data), "\n")

	w := bufio.NewWriter(out)
	defer w.Flush()
	for _, tok := range lexer.New(input).All() {
		// One token per line; literal newlines in token text become "\n" so
		// the line-oriented format stays unambiguous. EOF stays as "EOF".
		text := tok.Text
		if text != lexer.EOFText {
			text = strings.ReplaceAll(text, "\n", `\n`)
		}
		if _, err := fmt.Fprintln(w, text); err != nil {
			return err
		}
	}
	return nil
}
