package common

//go:generate mockgen -source get_now.go -destination __mock/get_now.go

import "time"

// GetNower is getter interface of now time.Time.
// Intention is to use as an unexported field of some structs.
// And make it mock-able inside internal tests.
type GetNower interface {
	GetNow() time.Time
}

type GetNowImpl struct {
}

// GetNow implements GetNower.
func (g GetNowImpl) GetNow() time.Time {
	return time.Now()
}
