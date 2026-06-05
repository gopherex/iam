package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/gopherex/iam/internal/domain"
)

// ErrorHandler renders request-level failures (parameter/body decode and the
// generated schema validation, plus security checks) into the same
// ErrorEnvelope as handler errors. These never reach Service.NewError — ogen
// raises them before the handler runs — so wire this with
// oas.WithErrorHandler(api.ErrorHandler) when building the server.
func ErrorHandler(_ context.Context, w http.ResponseWriter, _ *http.Request, err error) {
	de := classify(err)
	writeEnvelope(w, de)
}

// classify maps an ogen request-level error onto a domain error.
func classify(err error) *domain.Error {
	var de *domain.Error
	if errors.As(err, &de) {
		return de
	}
	var se *ogenerrors.SecurityError
	if errors.As(err, &se) {
		return domain.ErrUnauthorized
	}
	var (
		dre  *ogenerrors.DecodeRequestError
		dpse *ogenerrors.DecodeParamsError
		dpe  *ogenerrors.DecodeParamError
		dbe  *ogenerrors.DecodeBodyError
	)
	if errors.As(err, &dre) || errors.As(err, &dpse) || errors.As(err, &dpe) || errors.As(err, &dbe) {
		// Generated schema validation (minLength/pattern/format/required/…)
		// surfaces as a decode error.
		return domain.ErrValidation
	}
	return domain.ErrBadRequest
}

func writeEnvelope(w http.ResponseWriter, de *domain.Error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(de.Status)
	body := map[string]any{"error": map[string]any{"code": de.Code, "message": de.Message}}
	if len(de.Details) > 0 {
		body["error"].(map[string]any)["details"] = de.Details
	}
	_ = json.NewEncoder(w).Encode(body)
}
