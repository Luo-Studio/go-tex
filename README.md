# go-tex

A Go port of [RaTeX](https://github.com/erweixin/RaTeX), a KaTeX-compatible
LaTeX math rendering engine. The goal is byte-identical output against the
upstream Rust implementation while following idiomatic Go library practices.

## Status

| Stage | Score vs upstream | Notes |
|-------|-------------------|-------|
| Lex | **100%** (1099/1099) | byte-identical to upstream `lex` CLI |
| Parse | **61.78%** (679/1099) | byte-identical JSON AST vs upstream `parse` CLI |
| Parse, non-mhchem | **68.17%** (679/996) | mhchem (`\ce`, `\pu`) not yet ported |
| Layout / SVG / PNG | 0% | renderers not yet ported |

The "score" is a byte-identical text comparison against the upstream RaTeX
binaries, run line-by-line over the upstream golden corpus
(`testdata/golden/test_cases.txt` + `test_case_ce.txt`, 1099 inputs total).
Pixel-level comparison against the `output/`, `output_svg/` reference fixtures
is gated on porting the layout engine and renderers.

## Layout

```
go-tex/
├── cmd/
│   ├── lex/                  stdin -> one-token-per-line
│   └── parse/                stdin -> one-JSON-AST-per-line
├── tex/
│   ├── lexer/                LaTeX lexer + Token
│   ├── macroexp/             macro expander ("gullet")
│   ├── parser/               parser, AST, function registry
│   ├── symbols/              KaTeX symbols table (1876 entries)
│   ├── mathstyle/            TeX style transitions
│   ├── path/                 SVG path commands
│   └── source/               source-location helper
├── internal/parity/          test-time finder for upstream binaries
├── reference/ratex/          verbatim upstream Rust src (for porting refs)
└── testdata/golden/          upstream golden corpus
```

## What's implemented

**Lexer**: control words/symbols/space, catcodes (% comment, ~ active),
`\verb`/`\verb*`, combining-diacritical clustering, `@` in control words.
Full upstream test suite ported one-for-one; 100% byte parity on golden.

**Parser AST**: the upstream `ParseNode` enum mapped to Go structs. JSON
serialisation matches serde's `#[serde(tag = "type")]` shape exactly,
including KaTeX's CamelCase field names (`hasBarLine`, `isStretchy`,
`leftDelim`, …).

**Symbols table**: 1876 entries mechanically generated from
`symbols_data.rs`, with mode/codepoint lookup and the
`acceptUnicodeChar` fallback (so both `\alpha` and the literal `α`
resolve to the same MathOrd).

**Functions** (the parser dispatcher, growing): `\frac`, `\dfrac`,
`\tfrac`, `\cfrac`, `\binom`, `\dbinom`, `\tbinom`, `\sqrt`, math accents
(`\hat`, `\widehat`, `\tilde`, `\widetilde`, `\acute`, `\grave`, `\dot`,
`\ddot`, `\bar`, `\breve`, `\check`, `\vec`, `\mathring`, `\widecheck`,
the `\over*` arrow accents), under-accents, styling
(`\displaystyle`/`\textstyle`/`\scriptstyle`/`\scriptscriptstyle`), text
mode (`\text`, `\textrm`, `\textsf`, `\texttt`, `\textbf`, `\textit`, …),
math fonts (`\mathrm`, `\mathbf`, `\mathbb`, `\mathfrak`, `\Bbb`, `\frak`,
`\mathcal`, `\mathsf`, `\mathtt`, `\mathit`, …), `\left`/`\right`,
`\middle`, `\operatorname@`, `\operatornamewithlimits`, `\overline`,
`\underline`, `\phantom`/`\hphantom`/`\vphantom`, big operators (`\sum`,
`\int`, `\prod`, `\bigvee`, …) including limits/no-limits and
symbol/text variants and Unicode equivalents, `\mathop`, `\textcolor`,
`\color`, `\overbrace`/`\underbrace`, `\big`/`\Big`/`\bigl`/`\Bigr` and
all variants, infix `\over`/`\atop`/`\brace`/`\brack`/`\choose`/`\above`
with proper lowering to `GenFrac`, `\kern`/`\mkern`/`\hskip`/`\mskip` with
size-arg parsing.

**Macro expander**: token-stack gullet between lexer and parser; 238
built-in text-replacement macros generated from upstream
(Greek aliases, `\le`/`\ge`/`\dots`/etc.); `\TextOrMath` function macro
so the spacing kerns (`\!`, `\,`, `\:`, `\;`) reach the parser as the
right `\mskip{Nmu}` token stream.

**Not yet ported**: `\begin`/`\end` environments (matrix, array, align,
…), `\def`/`\gdef`/`\newcommand`/`\renewcommand`, arg-based macros
(`#1`, `#2`), mhchem (`\ce`, `\pu`), the layout engine, font loading,
SVG/PDF/PNG renderers, `\char`, `\@ifstar`, catcode changes, `\verbatim`.

## Running tests

```
go test ./...
```

The parser-parity tests need the upstream RaTeX binaries. If you have a
local RaTeX checkout:

```
cd path/to/RaTeX
cargo build --release -p ratex-lexer --bin lex
cargo build --release -p ratex-parser --bin parse
RATEX_TARGET_DIR=$(pwd)/target go test ./...
```

The tests skip cleanly if the binaries are unavailable.

## License

MIT — see `LICENSE`. Upstream RaTeX is also MIT.
