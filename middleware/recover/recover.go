package recover

import (
	"fmt"
	"time"

	"github.com/ngicks/gokugen"
)

type RecoveredError struct {
	OrginalErr any
}

func (re *RecoveredError) Error() string {
	return fmt.Sprintf(
		"recovered error: workFn panicked and recoved in Recover middleware. original err = %s",
		re.OrginalErr,
	)
}

type RecoverMiddleware struct{}

func New() *RecoverMiddleware {
	return &RecoverMiddleware{}
}

func (_ *RecoverMiddleware) Middleware(handler gokugen.ScheduleHandlerFn) gokugen.ScheduleHandlerFn {
	return func(ctx gokugen.SchedulerContext) (gokugen.Task, error) {
		return handler(
			gokugen.WrapContext(
				ctx,
				gokugen.WithWorkFnWrapper(
					func(self gokugen.SchedulerContext, workFn gokugen.WorkFn) gokugen.WorkFn {
						return func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) (ret any, err error) {
							defer func() {
								if recoveredErr := recover(); recoveredErr != nil {
									err = &RecoveredError{
										OrginalErr: recoveredErr,
									}
								}
							}()
							ret, err = workFn(ctxCancelCh, taskCancelCh, scheduled)
							return
						}
					},
				),
			),
		)
	}
}
