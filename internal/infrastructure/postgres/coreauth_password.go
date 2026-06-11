package postgres

// Password strength scoring used by CheckPassword. This is a self-contained
// heuristic — length tiers + character-class diversity + a common-password
// blocklist + obvious-pattern penalties — yielding a 0..4 score with the same
// scale zxcvbn uses. It is NOT the full zxcvbn dictionary/Markov model; it is an
// honest, dependency-free estimator so password_policy.zxcvbn_min_score gates on
// real signal (e.g. "Password1234" scores 0, not 4) rather than a two-rule toy.

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// commonPasswordBases are widely-breached password stems (lowercased, trailing
// digits/punctuation stripped before lookup). A match forces score 0.
var commonPasswordBases = map[string]struct{}{
	"password": {}, "passw0rd": {}, "qwerty": {}, "qwertyuiop": {}, "azerty": {},
	"admin": {}, "root": {}, "welcome": {}, "letmein": {}, "login": {},
	"abc": {}, "abcd": {}, "test": {}, "guest": {}, "user": {},
	"iloveyou": {}, "dragon": {}, "monkey": {}, "master": {}, "superman": {},
	"football": {}, "baseball": {}, "sunshine": {}, "princess": {}, "secret": {},
	"hello": {}, "whatever": {}, "trustno": {}, "changeme": {}, "default": {},
}

// passwordCharClasses counts the character classes present (lower, upper, digit,
// other) in pw.
func passwordCharClasses(pw string) int {
	var lower, upper, digit, other bool
	for _, r := range pw {
		switch {
		case unicode.IsLower(r):
			lower = true
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		default:
			other = true
		}
	}
	n := 0
	for _, ok := range []bool{lower, upper, digit, other} {
		if ok {
			n++
		}
	}
	return n
}

// passwordBase lowercases pw and strips a trailing run of digits and common
// suffix punctuation, so "Password1234!" reduces to "password" for blocklist
// lookup.
func passwordBase(pw string) string {
	b := strings.ToLower(strings.TrimSpace(pw))
	b = strings.TrimRight(b, "0123456789!@#$%^&*._-")
	return b
}

// hasObviousSequence reports a long monotonic run (abcd / 1234 / qwerty-row) or a
// single repeated character — both trivially guessable.
func hasObviousSequence(pw string) bool {
	if pw == "" {
		return false
	}
	lower := strings.ToLower(pw)
	for _, seq := range []string{"0123456789", "abcdefghijklmnopqrstuvwxyz", "qwertyuiop", "asdfghjkl", "zxcvbnm"} {
		if len(lower) >= 4 && strings.Contains(seq, lower) {
			return true
		}
	}
	// All-same character.
	first, _ := utf8.DecodeRuneInString(pw)
	allSame := true
	for _, r := range pw {
		if r != first {
			allSame = false
			break
		}
	}
	return allSame
}

// passwordStrengthScore returns a 0..4 estimate. 0 = trivially weak (too short,
// common, all one class & sequence), 4 = long and diverse.
func passwordStrengthScore(pw string) int {
	n := utf8.RuneCountInString(pw)
	if n == 0 {
		return 0
	}
	if _, common := commonPasswordBases[passwordBase(pw)]; common {
		return 0
	}
	score := 0
	switch {
	case n >= 16:
		score = 3
	case n >= 12:
		score = 2
	case n >= 10:
		score = 1
	case n >= 8:
		score = 1
	}
	// Character-class diversity bonus (beyond a single class).
	score += passwordCharClasses(pw) - 1
	// Penalize obvious sequences / single-char repeats.
	if hasObviousSequence(pw) {
		score -= 2
	}
	if score < 0 {
		score = 0
	}
	if score > 4 {
		score = 4
	}
	return score
}
