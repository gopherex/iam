package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/gopherex/iam/internal/domain"
)

// ErrorHandler renders generated-server failures (parameter/body decode,
// generated schema validation, security checks and response encoding) into the
// same ErrorEnvelope as handler errors. These never reach Service.NewError —
// ogen raises them around the handler — so wire this with
// oas.WithErrorHandler(api.ErrorHandler) when building the server.
func ErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	de := classify(err)
	logGeneratedError(ctx, r, de, err)
	writeEnvelope(w, de)
}

// classify maps an ogen/generated server error onto a domain error.
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
		return domain.ErrValidation.
			WithMessage("Request validation failed.").
			WithDetails(generatedErrorDetails("request_decode", err))
	}
	// Unknown errors here are not client input errors: in generated handlers this
	// path is also used when response validation/encoding fails after the handler
	// has returned data. Surface that as a server-side API contract failure.
	return domain.ErrInternal.
		WithMessage("Response validation failed. Check server logs.").
		WithDetails(generatedErrorDetails("response_encode", err))
}

type operationContext interface {
	OperationID() string
	OperationName() string
}

func generatedErrorDetails(stage string, err error) map[string]any {
	details := map[string]any{"stage": stage}
	if err != nil {
		details["error"] = err.Error()
	}
	var op operationContext
	if errors.As(err, &op) {
		if id := op.OperationID(); id != "" {
			details["operation_id"] = id
		}
		if name := op.OperationName(); name != "" {
			details["operation"] = name
		}
	}
	var dpe *ogenerrors.DecodeParamError
	if errors.As(err, &dpe) && dpe.Name != "" {
		details["param"] = dpe.Name
	}
	return details
}

func logGeneratedError(ctx context.Context, r *http.Request, de *domain.Error, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	level := slog.LevelWarn
	if de.Status >= http.StatusInternalServerError {
		level = slog.LevelError
	}
	attrs := []slog.Attr{
		slog.Int("status", de.Status),
		slog.String("code", de.Code),
	}
	if err != nil {
		attrs = append(attrs, slog.String("err", err.Error()))
	}
	if r != nil {
		attrs = append(attrs, slog.String("method", r.Method))
		if r.URL != nil {
			attrs = append(attrs, slog.String("path", r.URL.Path))
		}
	}
	var op operationContext
	if errors.As(err, &op) {
		if id := op.OperationID(); id != "" {
			attrs = append(attrs, slog.String("operation_id", id))
		}
		if name := op.OperationName(); name != "" {
			attrs = append(attrs, slog.String("operation", name))
		}
	}
	slog.LogAttrs(ctx, level, "api generated error", attrs...)
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
