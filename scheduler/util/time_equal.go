package util

import "time"

// TimeEqual determines equality of t and u, ignoring nano secs.
func TimeEqual(t, u time.Time) bool {
	return DropNanos(t).Equal(DropNanos(u))
}

func TimePointerEqual(t, u *time.Time, ignoreMilli bool) bool {
	if t == nil || u == nil {
		if t == nil && u == nil {
			return true
		}
		if t != nil || u != nil {
			return false
		}
	}
	if ignoreMilli {
		return TimeEqual(*t, *u)
	} else {
		return t.Equal(*u)
	}
}
