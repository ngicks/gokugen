package rescheduler

type Option func(r *Rescheduler)

func SetHook(hook ReschedulerHook) Option {
	return func(r *Rescheduler) {
		r.hook = hook
	}
}
