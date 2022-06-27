package common

//go:generate mockgen -source get_now.go -destination __mock/get_now.go

import "time"

type GetNow interface {
	GetNow() time.Time
}

type GetNowImpl struct {
}

func (g GetNowImpl) GetNow() time.Time {
	return time.Now()
}
