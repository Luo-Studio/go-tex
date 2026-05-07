# go-tex

A Go port of [RaTeX](https://github.com/erweixin/RaTeX), a KaTeX-compatible
LaTeX math rendering engine. The goal is byte-identical output against the
upstream Rust implementation while following idiomatic Go library practices.

## Status

| Stage | Score vs upstream RaTeX | Notes |
|-------|-------------------------|-------|
| Lex | **100%** (1099/1099) | byte-identical to upstream `lex` CLI on the full golden corpus |
| Parse, non-mhchem | **71.69%** (714/996) | canonical (JSON-roundtrip) match against upstream `parse` CLI |
| Parse, non-mhchem byte-identical | **68.37%** (681/996) | strict byte-identical match |
| Parse, all incl. mhchem | **64.97%** (714/1099) | mhchem `\ce`/`\pu` not yet ported |
| Layout / SVG / PNG | 0% | renderers not yet ported |

The headline score is **byte-equivalence** of textual outputs (token streams,
JSON ASTs) — not pixel-equivalence of rendered output. The two scores below
are reported by `go test ./tex/parser/... -run TestParseParityWithUpstream`
and are computed line-by-line over the upstream golden corpus
(`testdata/golden/test_cases.txt` + `test_case_ce.txt`, 1099 inputs total).
"Canonical" means both sides round-trip through `encoding/json` so trivial
formatting differences (whole-number floats, key ordering, whitespace) don't
artificially lower the score.

## Comparison to go-latex/latex

[`codeberg.org/go-latex/latex`](https://codeberg.org/go-latex/latex) is the
existing Go LaTeX library; per its README it is "NOT … a complete typesetting
system" and focuses narrowly on math equation rendering. go-tex aims to track
RaTeX's >99.5% KaTeX syntax coverage, so the parser already covers many
features that go-latex/latex does not (matrix family + align/aligned/cases/
gather/equation/array environments, `\begin/\end`, `\over/\atop/\above` infix
lowering, `\big/\Big/\bigl/\Bigr` delimiter sizing, `\TextOrMath`,
`\def/\gdef` with `#1`/`#2` argument substitution and source-position
preservation, macro nesting via the gullet, the full KaTeX symbols table
with `acceptUnicodeChar` fallback, …). The renderer is the next gap.

## Layout

```
go-tex/
├── cmd/
│   ├── lex/                  stdin -> one-token-per-line
│   └── parse/                stdin -> one-JSON-AST-per-line
├── tex/
│   ├── lexer/                LaTeX lexer + Token
│   ├── macroexp/             macro expander ("gullet")
│   ├── parser/               parser, AST, function registry, environments
│   ├── symbols/              KaTeX symbols table (1876 entries)
│   ├── mathstyle/            TeX style transitions
│   ├── path/                 SVG path commands
│   └── source/               source-location helper
├── internal/parity/          test-time finder for upstream binaries
├── reference/ratex/          verbatim upstream Rust src (porting reference)
└── testdata/golden/          upstream golden corpus
```

## What's implemented

**Lexer** (`tex/lexer`): control words/symbols/space, catcodes (`%` comment,
`~` active), `\verb`/`\verb*`, combining-diacritical clustering, `@` in
control words. Full upstream test suite ported one-for-one; **100% byte
parity** on the 1099-input golden corpus.

**Parser AST** (`tex/parser/ast.go`, `ast_more.go`): the upstream `ParseNode`
enum mapped to Go structs. JSON serialisation matches serde's
`#[serde(tag = "type")]` shape exactly, with KaTeX's CamelCase field names
(`hasBarLine`, `isStretchy`, `leftDelim`, `rowGaps`, `hLinesBeforeRow`, …).
`Measurement` emits `3.0` not `3` when whole, matching serde's f64 format.
`OrdGroup.Body` always serialises as `[]` (never `null`).

**Symbols table** (`tex/symbols`): 1876 entries mechanically generated from
`symbols_data.rs`, with mode/codepoint lookup and the `acceptUnicodeChar`
fallback so both `\alpha` and the literal `α` resolve to the same MathOrd.

**Macro expander** (`tex/macroexp`): token-stack gullet between lexer and
parser. 238 built-in text-replacement macros generated mechanically from
upstream (Greek aliases, `\le`/`\ge`/`\dots`/`\Bbbk`/…). `\TextOrMath`
function macro picks the right branch by mode, unlocking the spacing kerns
(`\!`, `\,`, `\:`, `\;`). `\def`/`\gdef`/`\edef`/`\xdef` parse a body as a
token slice (preserving original source positions) and substitute `#1`–`#9`
at the token level on expansion. `\@firstoftwo`, `\@secondoftwo`.

**Parser functions** (`tex/parser/functions*.go`): `\frac`, `\dfrac`, `\tfrac`,
`\cfrac`, `\binom`, `\dbinom`, `\tbinom`, `\sqrt`, math accents (`\hat`,
`\widehat`, `\tilde`, `\widetilde`, `\acute`, `\grave`, `\dot`, `\ddot`,
`\bar`, `\breve`, `\check`, `\vec`, `\mathring`, `\widecheck`, `\over*`
arrows), under-accents, styling (`\displaystyle`/`\textstyle`/`\scriptstyle`/
`\scriptscriptstyle`), text mode (`\text`, `\textrm`, `\textsf`, `\texttt`,
`\textbf`, `\textit`, …), math fonts (`\mathrm`, `\mathbf`, `\mathbb`,
`\mathfrak`, `\Bbb`, `\frak`, `\mathcal`, `\mathsf`, `\mathtt`, `\mathit`),
`\left`/`\right`/`\middle`, `\operatorname@`, `\operatornamewithlimits`,
`\overline`, `\underline`, `\phantom`/`\hphantom`/`\vphantom`, big operators
(`\sum`, `\int`, `\prod`, `\bigvee`, …) including limits/no-limits and
symbol/text variants and Unicode equivalents, `\mathop`, `\textcolor`,
`\color`, `\overbrace`/`\underbrace`, `\big`/`\Big`/`\bigl`/`\Bigr` and all
variants, infix `\over`/`\atop`/`\brace`/`\brack`/`\choose`/`\above` with
proper lowering to `GenFrac`, `\kern`/`\mkern`/`\hskip`/`\mskip`.

**Environments** (`tex/parser/environments.go`): `\begin{…}`/`\end{…}` for
the matrix family (`matrix`, `pmatrix`, `bmatrix`, `Bmatrix`, `vmatrix`,
`Vmatrix`, plus starred variants with `[l|c|r]` alignment), `smallmatrix`,
`cases`/`dcases`/`rcases`/`drcases`, `array`/`darray` (with column spec),
`align`/`aligned`/`split`/`alignat`/`alignedat` (with the upstream "prepend
empty ordgroup at every even-indexed cell" transform and the alternating
r-l column generation), `equation`/`equation*`, `gather`/`gathered`,
`subarray`.

**Not yet ported**: mhchem (`\ce`, `\pu` — ~104 cases), the CD environment,
`\newcommand`/`\renewcommand`/`\providecommand`, text-mode accents
(`\'`, `\"`, `\^`, `\~`, …), `\smash`/`\rule`/`\raisebox`/`\vcenter`/
`\includegraphics`/`\href`/`\url`, `\enclose` family (`\bcancel`, `\xcancel`,
`\sout`, `\colorbox`, `\fcolorbox`), `\char`/`\@ifstar`/`\noexpand`,
catcode changes, the layout engine, font loading, SVG/PDF/PNG renderers.

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
