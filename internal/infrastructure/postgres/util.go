package postgres

import "time"

// nowUTC is the current wall-clock time in UTC, used for updated_at columns.
func nowUTC() time.Time { return time.Now().UTC() }
