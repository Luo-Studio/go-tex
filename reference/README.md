# Upstream reference

This directory contains the upstream RaTeX source for the core crates we are
porting (types, lexer, parser, layout, svg, render, pdf), copied verbatim from
[erweixin/RaTeX](https://github.com/erweixin/RaTeX).

It is kept here purely so the Go port can iterate against the upstream
unit tests (the inline `#[cfg(test)]` blocks plus
`crates/ratex-parser/src/tests.rs`). Each Go test should have a clear lineage
to a Rust test we ported.

We deliberately do not include demos, platforms, fonts, scripts, or the
website. This copy is for reading only — it is not built and not part of the
Go module.
