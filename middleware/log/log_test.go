package log_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
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

	_, mockSched, getTrappedTask := mock_gokugen.BuildMockScheduler(t)

	ma := gokugen.NewMiddlewareApplicator(mockSched)

	inputCtx := gokugen.BuildContext(
		time.Now(),
		func(taskCtx context.Context, scheduled time.Time) (any, error) {
			return "baz", nil
		},
		nil,
		gokugen.WithTaskId("foo"),
		gokugen.WithWorkId("bar"),
	)
	inputError := gokugen.BuildContext(
		time.Now(),
		func(taskCtx context.Context, scheduled time.Time) (any, error) {
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
	getTrappedTask().Do(context.TODO())
	ma.Schedule(inputError)
	getTrappedTask().Do(context.TODO())
}

func TestLogOptionSetTimeFormat(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLogger := mock_log.NewMockLogger(ctrl)
	_, mockSched, getTrappedTask := mock_gokugen.BuildMockScheduler(t)

	pseudoScheduledTime := time.Date(2022, time.April, 12, 3, 12, 30, 0, time.UTC)

	inputCtx := gokugen.BuildContext(
		pseudoScheduledTime,
		func(taskCtx context.Context, scheduled time.Time) (any, error) {
			return "baz", nil
		},
		nil,
		gokugen.WithTaskId("foo"),
		gokugen.WithWorkId("bar"),
	)

	type TestCase struct {
		Format string
		Output string
	}
	testCases := []TestCase{
		{
			Format: time.ANSIC,
			Output: "Tue Apr 12 03:12:30 2022",
		},
		{
			Format: time.RFC822,
			Output: "12 Apr 22 03:12 UTC",
		},
		{
			Format: "ojlamglonlaoh;jhrgnmlj;ji;awo",
			Output: "ojlamglonlaoh;jhrgnmlj;ji;awo",
		},
	}
	for _, testCase := range testCases {
		mockLogger.
			EXPECT().
			Info(
				gomock.Any(),               // value
				gomock.Any(),               // "scheduled_at" label
				gomock.Eq(testCase.Output), // "scheduled_at" value
				gomock.Any(),               // "task_id" label
				gomock.Eq("foo"),           // "task_id" value
				gomock.Any(),               // "work_id" label
				gomock.Eq("bar"),           // "work_id" value
				gomock.Any(),               // "timing" label
				gomock.Any(),               // "timing" value
			).
			AnyTimes()

		ma := gokugen.NewMiddlewareApplicator(mockSched)

		logger := logmw.New(mockLogger, logmw.SetTimeFormat(testCase.Format))

		ma.Use(
			logger.Middleware,
		)

		ma.Schedule(inputCtx)
		getTrappedTask().Do(context.TODO())
	}
}

func TestLogOption(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLogger := mock_log.NewMockLogger(ctrl)
	_, mockSched, getTrappedTask := mock_gokugen.BuildMockScheduler(t)

	pseudoScheduledTime := time.Date(2022, time.April, 12, 3, 12, 30, 0, time.UTC)

	inputCtx := gokugen.BuildContext(
		pseudoScheduledTime,
		func(taskCtx context.Context, scheduled time.Time) (any, error) {
			return "baz", nil
		},
		map[any]any{
			"foo": "foofoo",
			"bar": true,
			"baz": 123,
			"qux": "quxqux",
		},
		gokugen.WithParam(map[string]string{
			"Foo": "foo",
			"Bar": "bar",
		}),
	)

	mapMarshalled, _ := json.Marshal(
		map[string]string{
			"Foo": "foo",
			"Bar": "bar",
		},
	)

	ma := gokugen.NewMiddlewareApplicator(mockSched)

	logger := logmw.New(
		mockLogger,
		logmw.LogParam(),
		logmw.LogAdditionalValues("foo", "bar", "baz"),
	)

	ma.Use(
		logger.Middleware,
	)

	mockLogger.
		EXPECT().
		Info(
			gomock.Any(), // value
			gomock.AssignableToTypeOf(reflect.TypeOf("")), // "scheduled_at" label
			gomock.AssignableToTypeOf(reflect.TypeOf("")), // "scheduled_at" value
			gomock.Eq("param"),                            // "param" label
			gomock.Eq(string(mapMarshalled)),              // "param" value
			gomock.Eq("foo"),                              // additionalKey label
			gomock.Eq("foofoo"),                           // additionalKey value
			gomock.Eq("bar"),                              // additionalKey label
			gomock.Eq("true"),                             // additionalKey value
			gomock.Eq("baz"),                              // additionalKey label
			gomock.Eq("123"),                              // additionalKey value
			gomock.AssignableToTypeOf(reflect.TypeOf("")), // "timing" label
			gomock.AssignableToTypeOf(reflect.TypeOf("")), // "timing" value
		).
		AnyTimes()

	ma.Schedule(inputCtx)
	getTrappedTask().Do(context.TODO())
}
