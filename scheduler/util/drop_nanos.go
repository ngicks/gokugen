// scheduler utility functions.
// Separated from scheduler to avoid circular module import.
package util

import "time"

// DropNanos drops nano seconds from t.
func DropNanos(t time.Time) time.Time {
	return t.Truncate(time.Millisecond)
}

func DropNanosPointer(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := DropNanos(*t)
	return &u
}
