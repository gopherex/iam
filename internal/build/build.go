package build

import (
	"fmt"

	"github.com/google/uuid"
)

var (
	ServiceName = "iam"     // set by -ldflags in release builds
	Version     = "0.0.0"   // set by -ldflags in release builds
	Commit      = "unknown" // set by -ldflags in release builds
	BuildTime   = "unknown" // RFC3339 UTC, set by -ldflags in release builds

	InstanceID = fmt.Sprintf("%s-%s-%s", ServiceName, Version, uuid.NewString())
)
