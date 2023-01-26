package rescheduler

import (
	"fmt"

	"github.com/ngicks/gokugen/scheduler"
)

type RuleNotFoundErr struct {
	Id string
}

func (e *RuleNotFoundErr) Error() string {
	return fmt.Sprintf("rule not found. id = %s", e.Id)
}

type Done struct {
	Id     string
	Reason string
}

func (e *Done) Error() string {
	return fmt.Sprintf("end of life. id = %s, reason = %s", e.Id, e.Reason)
}

type RuleNextErr struct {
	Id       string
	OldParam []byte
	Err      error
}

func (e *RuleNextErr) Error() string {
	return fmt.Sprintf(
		"rule next. id = %s, old param = %s, err = %+v",
		e.Id, string(e.OldParam), e.Err,
	)
}

type MetaUnmarshalErr struct {
	Err  error
	Task scheduler.Task
}

func (e *MetaUnmarshalErr) Error() string {
	return fmt.Sprintf("meta unmarshal error. err = %+v, task = %+v", e.Err, e.Task)
}
