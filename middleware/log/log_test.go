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
		Info(gomock.Eq("foo"), gomock.Eq("bar"), gomock.Eq("baz")).
		Times(1)
	mockLogger.
		EXPECT().
		Error(gomock.Eq("qux"), gomock.Eq("quux"), gomock.Eq(mockErr)).
		Times(1)

	_, mockSched, getTrappedTask := mock_gokugen.BuildMockScheduler(t)

	ma := gokugen.NewMiddlewareApplicator(mockSched)

	inputCtx :=
		gokugen.WithWorkId(
			gokugen.WithTaskId(
				gokugen.NewPlainContext(
					time.Now(),
					func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) (any, error) {
						return "baz", nil
					},
					nil,
				),
				"foo",
			),
			"bar",
		)
	inputError :=
		gokugen.WithWorkId(
			gokugen.WithTaskId(
				gokugen.NewPlainContext(
					time.Now(),
					func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) (any, error) {
						return "", mockErr
					},
					nil,
				),
				"qux",
			),
			"quux",
		)

	ma.Use(
		logmw.New(mockLogger).Middleware,
	)

	ma.Schedule(inputCtx)
	getTrappedTask().Do(make(<-chan struct{}))
	ma.Schedule(inputError)
	getTrappedTask().Do(make(<-chan struct{}))
}
