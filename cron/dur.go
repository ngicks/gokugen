package cron

import "time"

type Duration struct {
	Duration time.Duration `json:"duration"`
	Command  []string      `json:"command"`
}

func (d Duration) IsCommandValid() bool {
	return d.Command != nil && len(d.Command) != 0
}

func (d Duration) NextSchedule(now time.Time) (time.Time, error) {
	return now.Add(d.Duration), nil
}

func (d Duration) GetCommand() []string {
	if d.IsCommandValid() {
		return d.Command
	} else {
		return nil
	}
}
