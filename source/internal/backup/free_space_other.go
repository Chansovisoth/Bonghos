//go:build !linux

package backup

import "math"

// Bonghos production is Linux-only. Returning an effectively unlimited value
// keeps portable unit tests focused on backup behavior without pretending a
// non-Linux filesystem probe is production validation.
func storageFreeSpace(string) int64 { return math.MaxInt64 }
