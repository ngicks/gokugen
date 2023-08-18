package def

type ErrKindOption struct {
	SkipCancelledAt           bool
	SkipDispatchedAt          bool
	SkipDoneAt                bool
	ReturnOnEmptyDispatchedAt bool
}

func ErrKind(t Task, option ErrKindOption) RepositoryErrorKind {
	if !option.SkipDoneAt && t.DoneAt.IsSome() {
		return AlreadyDone
	}
	if !option.SkipCancelledAt && t.CancelledAt.IsSome() {
		return AlreadyCancelled
	}
	if !option.SkipDispatchedAt && t.DispatchedAt.IsSome() {
		return AlreadyDispatched
	}

	if option.ReturnOnEmptyDispatchedAt && t.DispatchedAt.IsNone() {
		return NotDispatched
	}

	return ""
}

func ErrKindUpdate(t Task) error {
	errKind := ErrKind(t, ErrKindOption{})
	if errKind != "" {
		return &RepositoryError{Id: t.Id, Kind: errKind}
	}
	return nil
}

func ErrKindCancel(t Task) error {
	errKind := ErrKind(t, ErrKindOption{})
	if errKind != "" {
		return &RepositoryError{Id: t.Id, Kind: errKind}
	}
	return nil
}

func ErrKindMarkAsDispatch(t Task) error {
	errKind := ErrKind(t, ErrKindOption{})
	if errKind != "" {
		return &RepositoryError{Id: t.Id, Kind: errKind}
	}
	return nil
}

func ErrKindMarkAsDone(t Task) error {
	errKind := ErrKind(t, ErrKindOption{
		ReturnOnEmptyDispatchedAt: true,
		SkipDispatchedAt:          true,
	})
	if errKind != "" {
		return &RepositoryError{Id: t.Id, Kind: errKind}
	}
	return nil
}
