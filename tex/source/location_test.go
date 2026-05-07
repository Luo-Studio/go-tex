package source

import "testing"

func TestRange(t *testing.T) {
	a := Location{Start: 0, End: 5}
	b := Location{Start: 10, End: 15}
	got := Range(a, b)
	want := Location{Start: 0, End: 15}
	if got != want {
		t.Errorf("Range(%v, %v) = %v, want %v", a, b, got, want)
	}
}
