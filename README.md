# go-tex

A Go port of [RaTeX](https://github.com/erweixin/RaTeX), a KaTeX-compatible LaTeX
math rendering engine. The goal is byte-identical golden output against the
upstream Rust implementation while following idiomatic Go library practices.

## Status

Work in progress. Currently implemented:

- `tex/source` — source location type shared across the pipeline.
- `tex/lexer` — LaTeX lexer producing the same token stream as RaTeX/KaTeX.
- `tex/mathstyle` — TeX math style transitions (display/text/script/scriptscript).
- `tex/path` — SVG-style path commands used by the renderer.
- `cmd/lex` — stdin → one-token-per-line CLI used by the lexer-compare harness.
- `testdata/golden` — copy of the upstream golden test corpus
  (fixtures, expected PNG and SVG output, test case inputs).

The parser, layout engine, font loading, and renderers are not yet ported.

## Layout

```
go-tex/
├── cmd/lex/                  CLI: lex stdin and write tokens to stdout
├── tex/
│   ├── lexer/                LaTeX lexer + token type
│   ├── mathstyle/            TeX math style transitions
│   ├── path/                 SVG-style path commands
│   └── source/               Source location helper
└── testdata/golden/          Upstream golden corpus (RaTeX tests/golden/)
```

## Running tests

```
go test ./...
```

## License

MIT — see `LICENSE` (to be added). Upstream RaTeX is also MIT licensed.
