package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/gopherex/iam/internal/domain"
)

func TestClassifyDecodeErrorIncludesActionableDetails(t *testing.T) {
	decodeErr := &ogenerrors.DecodeRequestError{
		OperationContext: ogenerrors.OperationContext{
			Name: "List admin users",
			ID:   "getV1ProjectsByProjectIdAdminUsers",
		},
		Err: errors.New("invalid environment header"),
	}

	de := classify(decodeErr)

	if de.Status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", de.Status, http.StatusUnprocessableEntity)
	}
	if de.Code != domain.ErrValidation.Code {
		t.Fatalf("code = %q, want %q", de.Code, domain.ErrValidation.Code)
	}
	if de.Message != "Request validation failed." {
		t.Fatalf("message = %q", de.Message)
	}
	assertDetail(t, de.Details, "stage", "request_decode")
	assertDetail(t, de.Details, "operation_id", "getV1ProjectsByProjectIdAdminUsers")
	assertDetail(t, de.Details, "operation", "List admin users")
	if got, _ := de.Details["error"].(string); !strings.Contains(got, "invalid environment header") {
		t.Fatalf("details.error = %q", got)
	}
}

func TestClassifyResponseEncodeErrorIsServerError(t *testing.T) {
	err := errors.New("validate: invalid: data.0.primary_email (string: no regex match)")

	de := classify(err)

	if de.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", de.Status, http.StatusInternalServerError)
	}
	if de.Code != domain.ErrInternal.Code {
		t.Fatalf("code = %q, want %q", de.Code, domain.ErrInternal.Code)
	}
	if de.Message != "Response validation failed. Check server logs." {
		t.Fatalf("message = %q", de.Message)
	}
	assertDetail(t, de.Details, "stage", "response_encode")
	if got, _ := de.Details["error"].(string); !strings.Contains(got, "primary_email") {
		t.Fatalf("details.error = %q", got)
	}
}

func TestServiceNewErrorPreservesDetails(t *testing.T) {
	de := domain.ErrValidation.
		WithMessage("Bad input.").
		WithDetails(map[string]any{
			"field": "email",
			"max":   254,
		})

	got := New().NewError(context.Background(), de)

	if got.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", got.StatusCode, http.StatusUnprocessableEntity)
	}
	if got.Response.Error.Code != domain.ErrValidation.Code {
		t.Fatalf("code = %q, want %q", got.Response.Error.Code, domain.ErrValidation.Code)
	}
	details, ok := got.Response.Error.Details.Get()
	if !ok {
		t.Fatal("details are not set")
	}
	var field string
	if err := json.Unmarshal(details["field"], &field); err != nil {
		t.Fatalf("decode field detail: %v", err)
	}
	if field != "email" {
		t.Fatalf("field detail = %q, want email", field)
	}
	var max int
	if err := json.Unmarshal(details["max"], &max); err != nil {
		t.Fatalf("decode max detail: %v", err)
	}
	if max != 254 {
		t.Fatalf("max detail = %d, want 254", max)
	}
}

func assertDetail(t *testing.T, details map[string]any, key, want string) {
	t.Helper()
	got, ok := details[key].(string)
	if !ok {
		t.Fatalf("details[%q] = %#v, want string", key, details[key])
	}
	if got != want {
		t.Fatalf("details[%q] = %q, want %q", key, got, want)
	}
}
