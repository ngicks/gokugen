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

// DropMicrosPointer returns new *Time which has same value as t but micro and nano seconds removed,
// when t is non-nil. It returns plain nil otherwise.
func DropMicrosPointer(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := DropMicros(*t)
	return &u
}
