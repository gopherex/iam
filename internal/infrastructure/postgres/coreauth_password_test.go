package postgres

import "testing"

func TestPasswordStrengthScore(t *testing.T) {
	cases := []struct {
		pw      string
		wantMax int // score must be <= this
		wantMin int // score must be >= this
	}{
		{"", 0, 0},
		{"password", 0, 0},         // common base
		{"Password1234", 0, 0},     // common base "password" after stripping digits
		{"qwerty", 0, 0},           // common
		{"aaaaaaaa", 0, 0},         // all-same repeat
		{"12345678", 0, 0},         // sequence + common-ish
		{"short", 0, 0},            // too short, single class
		{"correcthorse", 2, 1},     // long-ish, single class
		{"Tr0ub4dour&3xtra", 4, 3}, // long + diverse
		{"X9!qLm2@vBz7", 4, 3},     // 12 chars, 4 classes
	}
	for _, c := range cases {
		got := passwordStrengthScore(c.pw)
		if got < c.wantMin || got > c.wantMax {
			t.Errorf("score(%q) = %d, want [%d..%d]", c.pw, got, c.wantMin, c.wantMax)
		}
	}
}
