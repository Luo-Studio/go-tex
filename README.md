# go-tex

A Go port of [RaTeX](https://github.com/erweixin/RaTeX), a KaTeX-compatible
LaTeX math rendering engine. The goal is byte-identical output against the
upstream Rust implementation while following idiomatic Go library practices.

## Status

| Stage | Score vs upstream RaTeX | Notes |
|-------|-------------------------|-------|
| Lex | **100%** (1099/1099) | byte-identical to upstream `lex` CLI |
| Parse, non-mhchem | **97.89%** (975/996) | canonical match against upstream `parse` JSON AST |
| Parse, all incl. mhchem | **88.72%** (975/1099) | mhchem `\ce`/`\pu` deferred |
| **SVG byte parity** | **90.56%** (902/996) non-mhchem | byte-identical to upstream render-svg text mode |
| PNG render pipeline | working | oksvg+rasterx; AA differs from upstream's ab_glyph |
| PNG byte parity | 0% | requires bit-exact rasteriser matching ab_glyph |

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
│   ├── mathstyle/            TeX style transitions
│   ├── path/                 SVG path commands
│   ├── source/               source-location helper
│   ├── layout/               box layout engine + display list
│   ├── svg/                  display list -> SVG XML
│   ├── render/               SVG -> PNG via oksvg + rasterx
│   └── mhchem/               (skeleton) mhchem state machine port
├── internal/parity/          test-time finder for upstream binaries
├── reference/ratex/          verbatim upstream Rust src (porting reference)
└── testdata/golden/          upstream golden corpus
```

## Pipeline

```
LaTeX source
     │  tex/lexer (100% upstream parity)
     ▼
Tokens
     │  tex/macroexp + tex/parser (97.89% non-mhchem)
     ▼
ParseNode AST
     │  tex/layout + tex/fontmetrics
     ▼
LayoutBox tree
     │  tex/layout/displaylist (recursive emit with absolute positions)
     ▼
DisplayList
     │  tex/svg (text-mode SVG, 90.56% byte parity)
     ▼
SVG XML
     │  tex/render via oksvg + rasterx
     ▼
PNG bytes
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

- mhchem `\ce` / `\pu` (skeleton in tex/mhchem; engine/actions/texify
  pending — ~2000 lines of state-machine code).
- KaTeX stretchy SVG paths for `\widehat`/`\widetilde`/`\overrightarrow`
  /`\underbrace`/etc. and tall-delimiter paths for `\biggm\vert`,
  pmatrix-style auto-grown parens, etc. (~1200 lines in upstream's
  `katex_svg.rs`). These render as glyphs in the bundled fonts when the
  required size fits, otherwise upstream switches to per-pixel SVG paths
  that we don't generate yet.
- Full `\begin{CD}` commutative-diagram environment (parses to AST but
  renders blank cells).
- TTF glyph extraction for path-glyph SVG output (would let us match the
  upstream `output_svg/` golden corpus byte-for-byte, but requires a
  full sfnt parser and bezier extraction).
- PNG byte-parity vs upstream's ab_glyph rasteriser. The current
  oksvg+rasterx pipeline produces visually correct PNGs but with
  different anti-aliasing and hinting from upstream.

## License

MIT — see `LICENSE`. Upstream RaTeX is also MIT.
