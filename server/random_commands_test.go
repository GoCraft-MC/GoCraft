package server

import "testing"

func TestParseRandomRange(t *testing.T) {
	for _, test := range []struct {
		input        string
		minimum, max int64
		valid        bool
	}{
		{"1..6", 1, 6, true},
		{"-5..-2", -5, -2, true},
		{"6..1", 0, 0, false},
		{"1", 0, 0, false},
	} {
		minimum, maximum, err := parseRandomRange(test.input)
		if (err == nil) != test.valid || minimum != test.minimum || maximum != test.max {
			t.Fatalf("parseRandomRange(%q) = %d, %d, %v", test.input, minimum, maximum, err)
		}
	}
}
