package iambot

import (
	"strings"
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestRequestCard(t *testing.T) {
	t.Parallel()

	card := requestCard(&AccessRequest{ID: "ar_123", Email: "user@example.com", Reason: "need access"})
	for _, want := range []string{"user@example.com", "need access", "ar_123"} {
		if !strings.Contains(card, want) {
			t.Fatalf("card missing %q: %q", want, card)
		}
	}

	if strings.Contains(card, "Reason:") && !strings.Contains(card, "need access") {
		t.Fatal("reason label without body")
	}

	// No reason → no dangling label.
	if plain := requestCard(&AccessRequest{ID: "ar_1", Email: "a@b.c"}); strings.Contains(plain, "Reason") {
		t.Fatalf("card without reason must not carry the label: %q", plain)
	}
}

func TestDecisionKeyboard(t *testing.T) {
	t.Parallel()

	kb := decisionKeyboard("ar_123")
	if len(kb.InlineKeyboard) != 1 || len(kb.InlineKeyboard[0]) != 2 {
		t.Fatalf("keyboard layout = %+v", kb.InlineKeyboard)
	}

	approve := kb.InlineKeyboard[0][0]
	deny := kb.InlineKeyboard[0][1]

	if approve.CallbackData != "ar:app:ar_123" || deny.CallbackData != "ar:deny:ar_123" {
		t.Fatalf("callback data = %q / %q", approve.CallbackData, deny.CallbackData)
	}

	// Telegram caps callback data at 64 bytes; prefix + uuid fits.
	if len(approve.CallbackData) > 64 {
		t.Fatalf("callback data too long for telegram: %d", len(approve.CallbackData))
	}
}

func TestShortID(t *testing.T) {
	t.Parallel()

	if got := shortID("abcdefghijklmnop"); got != "abcdefgh" {
		t.Fatalf("shortID = %q", got)
	}

	if got := shortID("abc"); got != "abc" {
		t.Fatalf("shortID short = %q", got)
	}
}

func TestConfigAllows(t *testing.T) {
	t.Parallel()

	cfg := Config{AllowedUsers: []int64{42, 7}}
	if !cfg.Allows(42) || !cfg.Allows(7) || cfg.Allows(43) {
		t.Fatal("allowlist membership wrong")
	}
}

func TestUserIDExtraction(t *testing.T) {
	t.Parallel()

	u := &models.Update{Message: &models.Message{From: &models.User{ID: 11}}}
	if userID(u) != 11 {
		t.Fatalf("message user = %d", userID(u))
	}

	u = &models.Update{CallbackQuery: &models.CallbackQuery{From: models.User{ID: 22}}}
	if userID(u) != 22 {
		t.Fatalf("callback user = %d", userID(u))
	}

	if userID(&models.Update{}) != 0 {
		t.Fatal("empty update must yield 0")
	}
}
