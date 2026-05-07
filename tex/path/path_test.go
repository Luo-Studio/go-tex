package path

import "testing"

func TestConstructors(t *testing.T) {
	tests := []struct {
		name string
		got  Command
		want Command
	}{
		{"MoveTo", MoveTo(1, 2), Command{Kind: KindMoveTo, X: 1, Y: 2}},
		{"LineTo", LineTo(3, 4), Command{Kind: KindLineTo, X: 3, Y: 4}},
		{"CubicTo", CubicTo(1, 2, 3, 4, 5, 6), Command{Kind: KindCubicTo, X1: 1, Y1: 2, X2: 3, Y2: 4, X: 5, Y: 6}},
		{"QuadTo", QuadTo(1, 2, 3, 4), Command{Kind: KindQuadTo, X1: 1, Y1: 2, X: 3, Y: 4}},
		{"Close", Close(), Command{Kind: KindClose}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %+v, want %+v", tc.got, tc.want)
			}
		})
	}
}
