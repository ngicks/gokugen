// scheduler utility functions.
// Separated from scheduler to avoid circular module import.
package util

import "time"

// DropMicros drops micro and nano seconds from t.
func DropMicros(t time.Time) time.Time {
	// TODO: use more specialized implementation?
	//```go
	// t.Add(-time.Duration(t.Nanosecond() % 1e6))
	//```
	// This is basically same as t.Truncate but less if's and switch's.
	// But above impl does not remove a monotonic clock record.
	return t.Truncate(time.Millisecond)
}
