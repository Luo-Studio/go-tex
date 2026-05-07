//! PDF export for RaTeX [`DisplayList`](ratex_types::display_item::DisplayList).
//!
//! Built directly on [pdf-writer](https://docs.rs/pdf-writer) with manual font subsetting
//! via [subsetter](https://docs.rs/subsetter). Produces compact PDFs with embedded CIDFontType2
//! fonts and `Identity-H` encoding. No high-level abstraction layer.
//!
//! # Font sources
//!
//! - **Without** the `embed-fonts` feature: set [`PdfOptions::font_dir`] to a directory that
//!   contains the KaTeX `.ttf` files (same layout as the repo `fonts/` tree). The default
//!   [`PdfOptions::default`] uses an empty `font_dir` and will fail at render time until you set it.
//! - **With** `embed-fonts`: glyph bytes are loaded from the `ratex-katex-fonts` crate; `font_dir` is
//!   ignored. The `render-pdf` binary (`cli` feature) always enables `embed-fonts`, so its
//!   `--font-dir` flag does not affect which fonts are embedded.

mod fonts;
mod renderer;

pub use renderer::{render_to_pdf, PdfError, PdfOptions};
