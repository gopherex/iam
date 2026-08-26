package postgres

import (
	"context"
	"fmt"

	"github.com/go-faster/jx"

	"github.com/gopherex/iam/internal/domain"
)

// Retention policy is stored as a project-level iam_config doc (key=retention).
// The GC worker reads it to prune audit logs and events past the configured
// window (see gc.go). The policy is round-tripped as a raw JSON object so the
// handler owns the typed shape (oas.RetentionPolicy).

const retentionConfigKey = "retention"

// GetRetentionPolicy returns the stored policy as a raw JSON object ("{}" when
// unset).
func (a *pgAdminConfig) GetRetentionPolicy(ctx context.Context, projectID string) ([]byte, error) {
	doc, err := a.getConfigDoc(ctx, projectID, "", retentionConfigKey)
	if err != nil {
		return nil, err
	}

	return configDocToRawJSON(doc)
}

// PutRetentionPolicy replaces the policy from a raw JSON object.
func (a *pgAdminConfig) PutRetentionPolicy(ctx context.Context, projectID string, raw []byte) error {
	doc, err := rawJSONToConfigDoc(raw)
	if err != nil {
		return domain.ErrValidation.WithMessage("retention policy must be a JSON object")
	}

	_, err = a.putConfigDoc(ctx, projectID, "", retentionConfigKey, doc)

	return err
}

// rawJSONToConfigDoc parses a flat JSON object into an AdminConfigDoc (top-level
// keys -> raw values), the inverse of configDocToRawJSON.
func rawJSONToConfigDoc(raw []byte) (domain.AdminConfigDoc, error) {
	doc := domain.AdminConfigDoc{}
	if len(raw) == 0 {
		return doc, nil
	}

	d := jx.DecodeBytes(raw)

	err := d.Obj(func(d *jx.Decoder, key string) error {
		v, err := d.Raw()
		if err != nil {
			return fmt.Errorf("decode retention policy field %q: %w", key, err)
		}

		doc[key] = v

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("decode retention policy: %w", err)
	}

	return doc, nil
}
