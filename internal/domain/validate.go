package domain

import "strings"

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
