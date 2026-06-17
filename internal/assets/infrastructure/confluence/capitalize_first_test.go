package confluence

import "testing"

func TestCapitalizeFirst(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},                             // previously-uncovered empty short-circuit
		{"a", "A"},                           // single character
		{"hello", "Hello"},                   // ordinary word
		{"Hello", "Hello"},                   // already capitalized
		{" leading space", " leading space"}, // first byte is a space; ToUpper is identity
		{"1number", "1number"},               // first byte is a digit
	}
	for _, c := range cases {
		if got := capitalizeFirst(c.in); got != c.want {
			t.Errorf("capitalizeFirst(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
