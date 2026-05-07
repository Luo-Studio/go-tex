# go-tex

A Go port of [RaTeX](https://github.com/erweixin/RaTeX), a KaTeX-compatible
LaTeX math rendering engine. The goal is byte-identical output against the
upstream Rust implementation while following idiomatic Go library practices.

## Status

| Stage | Score vs upstream RaTeX | Notes |
|-------|-------------------------|-------|
| Lex | **100%** (1099/1099) | byte-identical to upstream `lex` CLI |
| Parse, non-mhchem | **97.89%** (975/996) | canonical match against upstream `parse` JSON AST |
| Parse, non-mhchem byte-identical | **91.37%** (910/996) | strict byte-for-byte |
| Parse, all incl. mhchem | **88.72%** (975/1099) | mhchem `\ce`/`\pu` not yet ported |
| Layout / SVG / PNG | 0% | renderers not yet ported |

The headline score is **byte-equivalence of textual outputs** (token streams,
JSON ASTs) — not pixel-equivalence of rendered output. Pixel comparison
against the `output*/` reference fixtures is gated on porting the layout
engine and renderers.

## Comparison to go-latex/latex

[`codeberg.org/go-latex/latex`](https://codeberg.org/go-latex/latex) is the
existing Go LaTeX library; per its README it is "NOT … a complete typesetting
system" and focuses narrowly on math equation rendering. go-tex aims to
track RaTeX's >99.5% KaTeX syntax coverage. Currently the parser already
covers the entire matrix/align/cases/array/CD/equation/gather/subarray
environment family, the full set of accents and styling commands, the
operator/big-operator family with `\limits`/`\nolimits`, `\big`/`\Big`/etc.
delimiter sizing, infix `\over`/`\atop`/`\above` lowering, `\textcolor`,
`\colorbox`/`\fcolorbox`, `\overbrace`/`\underbrace`, `\overbracket`/
`\underbracket`, `\href`/`\url`, `\smash`, `\rule`, `\raisebox`,
`\mathchoice`, `\mathord`/`\mathbin`/`\mathrel`/`\mathopen`/`\mathclose`/
`\mathpunct`/`\mathinner`, `\stackrel`/`\overset`/`\underset`,
`\xrightarrow`/`\xleftarrow`-family, `\boldsymbol`/`\bm`, old-style font
switches `\rm`/`\bf`/`\it`/`\sf`/`\tt`/`\cal`, sizing (`\tiny`…`\Huge`),
`\hbox`/`\fbox`/`\angl`, `\@char`, `\html@mathml`, math-mode switches
`\(`/`\)`/`$`, the full KaTeX symbols table with `acceptUnicodeChar`
fallback (1876 entries), the macro gullet (`\def`/`\gdef`/`\edef`/`\xdef`,
`\newcommand`/`\renewcommand`/`\providecommand` with `#1`–`#9` substitution
and source-position preservation, `\@ifstar`, `\@ifnextchar`,
`\@firstoftwo`/`\@secondoftwo`, `\char`, `\noexpand`, html-pass macros,
`\TextOrMath`, 240 built-in text-replacement macros), `\arraystretch`
honoured at array-build time, equation/align/gather auto-numbering,
`\hline`/`\hdashline` tracking.

## Layout

```
go-tex/
├── cmd/
│   ├── lex/                  stdin -> one-token-per-line
│   └── parse/                stdin -> one-JSON-AST-per-line
├── tex/
│   ├── lexer/                LaTeX lexer + Token
│   ├── macroexp/             macro expander (the "gullet")
│   ├── parser/               parser, AST, function registry, environments
│   ├── symbols/              KaTeX symbols table (1876 entries)
│   ├── mathstyle/            TeX style transitions
│   ├── path/                 SVG path commands
│   └── source/               source-location helper
├── internal/parity/          test-time finder for upstream binaries
├── reference/ratex/          verbatim upstream Rust src (porting reference)
└── testdata/golden/          upstream golden corpus
```

## Not yet ported

- **mhchem** (`\ce`, `\pu`) — the chemistry / physical-units parser. ~104
  cases in the corpus depend on it. Upstream has a 2000-line state-machine
  port from KaTeX's mhchem.js, plus 170 KB of state-machine JSON data. This
  is the largest remaining gap.
- **`\edef` (immediate expansion)** — currently behaves like `\def`. 3
  cases use the immediate-expansion semantics.
- **`\genfrac`** — full 6-arg generalised fraction. The convenience forms
  `\frac`/`\binom`/`\dfrac`/`\dbinom`/`\tfrac`/`\tbinom`/`\cfrac` work.
- The layout engine, font loading, SVG/PDF/PNG renderers.

## Running tests

```
go test ./...
```

The parity tests need the upstream RaTeX binaries. With a local RaTeX
checkout:

```
cd path/to/RaTeX
cargo build --release -p ratex-lexer --bin lex
cargo build --release -p ratex-parser --bin parse
RATEX_TARGET_DIR=$(pwd)/target go test ./...
```

The tests skip cleanly if the binaries are unavailable.

## License

MIT — see `LICENSE`. Upstream RaTeX is also MIT.
