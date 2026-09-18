package store

import "errors"

// ErrNotFound is returned when a fighter id does not exist or is incomplete
// (no current division / no fights).
var ErrNotFound = errors.New("fighter not found")
