package log_test

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ngicks/gokugen"
	mock_gokugen "github.com/ngicks/gokugen/__mock"
	logmw "github.com/ngicks/gokugen/middleware/log"
	mock_log "github.com/ngicks/gokugen/middleware/log/__mock"
)

func TestLog(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLogger := mock_log.NewMockLogger(ctrl)

	mockErr := errors.New("mock error")
	mockLogger.
		EXPECT().
		Info(
			gomock.Any(), // value
			gomock.Any(), // "scheduled_at" label
			gomock.Any(), // "scheduled_at" value
			gomock.Any(), // "task_id" label
			gomock.Any(), // "task_id" value
			gomock.Any(), // "work_id" label
			gomock.Any(), // "work_id" value
			gomock.Any(), // "timing" label
			gomock.Any(), // "timing" value
		).
		// 2 if successful, 1 if error
		Times(3)
	mockLogger.
		EXPECT().
		Error(
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
		).
		Times(1)

	_,
		mockSched, getTrappedTask := mock_gokugen.BuildMockScheduler(t)

	ma := gokugen.NewMiddlewareApplicator(mockSched)

	inputCtx := gokugen.BuildContext(
		time.Now(),
		func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) (any, error) {
			return "baz", nil
		},
		nil,
		gokugen.WithTaskId("foo"),
		gokugen.WithWorkId("bar"),
	)
	inputError := gokugen.BuildContext(
		time.Now(),
		func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) (any, error) {
			return "", mockErr
		},
		nil,
		gokugen.WithTaskId("qux"),
		gokugen.WithWorkId("quux"),
	)

	ma.Use(
		logmw.New(mockLogger).Middleware,
	)

	ma.Schedule(inputCtx)
	getTrappedTask().Do(make(<-chan struct{}))
	ma.Schedule(inputError)
	getTrappedTask().Do(make(<-chan struct{}))
}
