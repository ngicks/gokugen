package cron

import "time"

// Duration implements same RowLike interface as Row.
type Duration struct {
	Duration time.Duration `json:"duration"`
	Command  []string      `json:"command"`
}

func (d Duration) IsCommandValid() bool {
	return d.Command != nil && len(d.Command) != 0
}

// NextSchedule returns now + Duration.
func (d Duration) NextSchedule(now time.Time) (time.Time, error) {
	return now.Add(d.Duration), nil
}

// GetCommand returns []string if Command is valid, nil otherwise.
// Mutating returned slice causes undefined behavior.
func (d Duration) GetCommand() []string {
	if d.IsCommandValid() {
		return d.Command
	} else {
		return nil
	}
}
