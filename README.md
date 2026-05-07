# go-tex

A Go port of [RaTeX](https://github.com/erweixin/RaTeX), a KaTeX-compatible
LaTeX math rendering engine. The goal is byte-identical output against the
upstream Rust implementation while following idiomatic Go library practices.

## Status

| Stage | Score vs upstream RaTeX | Notes |
|-------|-------------------------|-------|
| Lex | **100%** (1099/1099) | byte-identical to upstream `lex` CLI |
| Parse, non-mhchem | **99.00%** (986/996) | byte-identical against upstream `parse` JSON AST |
| Parse, mhchem (`\ce`, `\pu`) | **100%** (103/103) | full state-machine port, byte-identical |
| **Parse, overall** | **99.09%** (1089/1099) | combined |
| SVG byte parity, non-mhchem | **99.00%** (986/996) | byte-identical to upstream render-svg text mode |
| SVG byte parity, mhchem | **98.06%** (101/103) | |
| PNG render pipeline | working | tdewolff/canvas + embedded KaTeX TTFs |
| PDF render pipeline | working | go-pdf/fpdf + embedded KaTeX TTFs |

The headline score is **byte-equivalence of textual outputs** (token streams,
JSON ASTs, SVG XML) rather than pixel-equivalence of rasterised PNGs. Pixel
parity is much harder because it requires byte-exact raster algorithms; text
parity is far more achievable.

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
│   ├── fontmetrics/          KaTeX font metrics tables (19 fonts)
│   ├── fonts/                embedded KaTeX TTFs + loader
│   ├── mathstyle/            TeX style transitions
│   ├── path/                 SVG path commands
│   ├── source/               source-location helper
│   ├── layout/               box layout engine + display list
│   ├── svg/                  display list -> SVG XML
│   ├── canvasr/              display list -> tdewolff/canvas (PNG/SVG/EPS)
│   ├── render/               display list -> PNG via canvasr + rasterizer
│   ├── pdf/                  display list -> PDF via codeberg.org/go-pdf/fpdf
│   └── mhchem/               KaTeX mhchem 3.3.0 (\ce, \pu) state machine
├── internal/parity/          test-time finder for upstream binaries
└── testdata/golden/          upstream golden corpus
```

## Pipeline

```
LaTeX source
     │  tex/lexer (100% upstream parity)
     ▼
Tokens
     │  tex/macroexp + tex/parser (99.00% non-mhchem, 100% mhchem)
     ▼
ParseNode AST
     │  tex/layout + tex/fontmetrics
     ▼
LayoutBox tree
     │  tex/layout/displaylist (recursive emit with absolute positions)
     ▼
DisplayList
     │       ├─ tex/svg (text-mode SVG, 99.00% byte parity)
     │       │       ▼
     │       │  SVG XML
     │       │
     │       ├─ tex/render (PNG via tdewolff/canvas + embedded KaTeX TTFs)
     │       │       ▼
     │       │  PNG bytes
     │       │
     │       └─ tex/pdf (PDF via go-pdf/fpdf + embedded KaTeX TTFs)
     │               ▼
     │          PDF bytes
```

## Comparison to go-latex/latex

[`codeberg.org/go-latex/latex`](https://codeberg.org/go-latex/latex) is the
existing Go LaTeX library; per its README it is "NOT … a complete typesetting
system". go-tex now has **end-to-end PNG rendering** for math expressions
through pure Go (no native deps), with parser coverage matching upstream
RaTeX's >99% KaTeX support.

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
cargo build --release -p ratex-layout
cargo build --release -p ratex-svg --features cli,standalone --bin render-svg
RATEX_TARGET_DIR=$(pwd)/target go test ./...
```

The tests skip cleanly if the binaries are unavailable.

## Not yet ported

- 10 cases (1.00%) of non-mhchem parse parity remain. Mostly individual
  function-shape mismatches; mhchem itself is 100%.
- A few cases of SVG byte parity are floating-point precision drift in
  cubic Bezier flattening — the underlying f64 values differ by 1 ULP
  from upstream Rust at certain points, even with bit-identical formulas.
  Visually indistinguishable.
- TTF glyph extraction for path-glyph SVG output (would let us match the
  upstream `output_svg/` golden corpus byte-for-byte, but requires a
  full sfnt parser and bezier extraction).
- PNG byte-parity vs upstream's ab_glyph rasteriser. The current
  oksvg+rasterx pipeline produces visually correct PNGs but with
  different anti-aliasing and hinting from upstream.

## License

MIT — see `LICENSE`. Upstream RaTeX is also MIT.
