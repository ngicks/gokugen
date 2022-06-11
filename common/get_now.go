package common

import "time"

type GetNow interface {
	GetNow() time.Time
}

type GetNowImpl struct {
}

func (g GetNowImpl) GetNow() time.Time {
	return time.Now()
}
