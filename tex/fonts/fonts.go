// Package fonts embeds the KaTeX TTF fonts and provides a
// loader keyed by the layout package's font ID strings ("Main-Regular",
// "Math-Italic", "Size2-Regular", etc.).
//
// KaTeX fonts are © Copyright 2014–2020 Khan Academy, licensed under the
// SIL Open Font License Version 1.1. See data/LICENSE.
package fonts

import (
	"embed"
	"fmt"
	"sync"
)

//go:embed data/*.ttf
var ttfFS embed.FS

// fontFiles maps a layout-package font ID to the TTF filename in data/.
var fontFiles = map[string]string{
	"Main-Regular":         "KaTeX_Main-Regular.ttf",
	"Main-Bold":            "KaTeX_Main-Bold.ttf",
	"Main-Italic":          "KaTeX_Main-Italic.ttf",
	"Main-BoldItalic":      "KaTeX_Main-BoldItalic.ttf",
	"Math-Italic":          "KaTeX_Math-Italic.ttf",
	"Math-BoldItalic":      "KaTeX_Math-BoldItalic.ttf",
	"AMS-Regular":          "KaTeX_AMS-Regular.ttf",
	"Caligraphic-Regular":  "KaTeX_Caligraphic-Regular.ttf",
	"Caligraphic-Bold":     "KaTeX_Caligraphic-Bold.ttf",
	"Fraktur-Regular":      "KaTeX_Fraktur-Regular.ttf",
	"Fraktur-Bold":         "KaTeX_Fraktur-Bold.ttf",
	"SansSerif-Regular":    "KaTeX_SansSerif-Regular.ttf",
	"SansSerif-Bold":       "KaTeX_SansSerif-Bold.ttf",
	"SansSerif-Italic":     "KaTeX_SansSerif-Italic.ttf",
	"Script-Regular":       "KaTeX_Script-Regular.ttf",
	"Typewriter-Regular":   "KaTeX_Typewriter-Regular.ttf",
	"Size1-Regular":        "KaTeX_Size1-Regular.ttf",
	"Size2-Regular":        "KaTeX_Size2-Regular.ttf",
	"Size3-Regular":        "KaTeX_Size3-Regular.ttf",
	"Size4-Regular":        "KaTeX_Size4-Regular.ttf",
}

var (
	cacheMu sync.RWMutex
	cache   = map[string][]byte{}
)

// TTF returns the raw TTF bytes for a layout-package font ID. Returns
// nil + error if the ID is unknown or the embedded file is missing.
func TTF(fontID string) ([]byte, error) {
	cacheMu.RLock()
	if b, ok := cache[fontID]; ok {
		cacheMu.RUnlock()
		return b, nil
	}
	cacheMu.RUnlock()
	name, ok := fontFiles[fontID]
	if !ok {
		return nil, fmt.Errorf("fonts: unknown font ID %q", fontID)
	}
	b, err := ttfFS.ReadFile("data/" + name)
	if err != nil {
		return nil, fmt.Errorf("fonts: read %s: %w", name, err)
	}
	cacheMu.Lock()
	cache[fontID] = b
	cacheMu.Unlock()
	return b, nil
}

// Names returns the list of supported font IDs in deterministic order.
func Names() []string {
	out := make([]string, 0, len(fontFiles))
	for k := range fontFiles {
		out = append(out, k)
	}
	return out
}
