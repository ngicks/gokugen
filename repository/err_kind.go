package repository

import "github.com/ngicks/gokugen/scheduler"

type ErrKindOption struct {
	SkipCancelledAt           bool
	SkipDispatchedAt          bool
	SkipDoneAt                bool
	ReturnOnEmptyDispatchedAt bool
}

func ErrKind(t scheduler.Task, option ErrKindOption) scheduler.RepositoryErrorKind {
	if !option.SkipDoneAt && t.DoneAt != nil {
		return scheduler.AlreadyDone
	}
	if !option.SkipCancelledAt && t.CancelledAt != nil {
		return scheduler.AlreadyCancelled
	}
	if !option.SkipDispatchedAt && t.DispatchedAt != nil {
		return scheduler.AlreadyDispatched
	}

	if option.ReturnOnEmptyDispatchedAt && t.DispatchedAt == nil {
		return scheduler.NotDispatched
	}

	return ""
}

func ErrKindUpdate(t scheduler.Task) error {
	errKind := ErrKind(t, ErrKindOption{})
	if errKind != "" {
		return &scheduler.RepositoryError{Id: t.Id, Kind: errKind}
	}
	return nil
}

func ErrKindCancel(t scheduler.Task) error {
	errKind := ErrKind(t, ErrKindOption{
		SkipCancelledAt: true,
	})
	if errKind != "" {
		return &scheduler.RepositoryError{Id: t.Id, Kind: errKind}
	}
	return nil
}

func ErrKindMarkAsDispatch(t scheduler.Task) error {
	errKind := ErrKind(t, ErrKindOption{})
	if errKind != "" {
		return &scheduler.RepositoryError{Id: t.Id, Kind: errKind}
	}
	return nil
}

func ErrKindMarkAsDone(t scheduler.Task) error {
	errKind := ErrKind(t, ErrKindOption{
		ReturnOnEmptyDispatchedAt: true,
		SkipDispatchedAt:          true,
	})
	if errKind != "" {
		return &scheduler.RepositoryError{Id: t.Id, Kind: errKind}
	}
	return nil
}
