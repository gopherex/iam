package domain

import (
	"regexp"
	"strings"
)

// Command validations are business rules BEYOND the wire-schema checks ogen
// runs on decode (length/pattern/required). They return a domain *Error so the
// API layer renders them like any other domain failure.

// Validate checks sign-up invariants: an identifier is required.
func (c RegisterCmd) Validate() error {
	if strings.TrimSpace(c.Email) == "" && strings.TrimSpace(c.Phone) == "" {
		return ErrValidation.WithMessage("email or phone is required")
	}
	return nil
}

// Validate checks profile-update invariants.
func (c ProfileUpdateCmd) Validate() error {
	if c.AccountID == "" || c.ProjectID == "" {
		return ErrValidation.WithMessage("project and account are required")
	}
	return nil
}

// Validate checks project-create invariants.
func (c ProjectCmd) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return ErrValidation.WithMessage("project name is required")
	}
	return nil
}

// Validate checks connection-create invariants.
func (c ConnectionCmd) Validate() error {
	if c.ProjectID == "" || c.Type == "" {
		return ErrValidation.WithMessage("project and connection type are required")
	}
	return nil
}

var phoneE164Re = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

// ValidatePhone checks that phone matches E.164 format (starts with +, 7-15
// digits after the plus sign, no leading zero on the country code).
func ValidatePhone(phone string) error {
	if !phoneE164Re.MatchString(phone) {
		return ErrValidation.WithMessage("phone must be in E.164 format (+<country><number>, 7-15 digits)")
	}
	return nil
}

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// ValidateEmail checks basic email format (contains @, has a domain part with a
// dot). This is NOT a full RFC 5322 check — it rejects obviously malformed
// addresses without false-positive-rejecting valid ones.
func ValidateEmail(email string) error {
	if !emailRe.MatchString(email) {
		return ErrValidation.WithMessage("email format is invalid")
	}
	return nil
}
