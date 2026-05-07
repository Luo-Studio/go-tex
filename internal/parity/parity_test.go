package parity

import "testing"

func TestEnvVarFor(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"lex", "RATEX_LEX"},
		{"render-svg", "RATEX_RENDER_SVG"},
	}
	for _, c := range cases {
		if got := envVarFor(c.in); got != c.want {
			t.Errorf("envVarFor(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
