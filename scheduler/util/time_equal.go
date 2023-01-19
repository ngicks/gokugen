package util

import "time"

// TimeEqual determines equality of t and u, ignoring micro and nano secs.
func TimeEqual(t, u time.Time) bool {
	return DropMicros(t).Equal(DropMicros(u))
}

func TimePointerEqual(t, u *time.Time, ignoreMicro bool) bool {
	if t == nil || u == nil {
		if t == nil && u == nil {
			return true
		}
		if t != nil || u != nil {
			return false
		}
	}
	if ignoreMicro {
		return TimeEqual(*t, *u)
	} else {
		return t.Equal(*u)
	}
}
