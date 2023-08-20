package def

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrKind(t *testing.T) {
	assert := assert.New(t)

	type testCase struct {
		task Task
		opt  ErrKindOption
		kind RepositoryErrorKind
	}

	for _, tc := range []testCase{
		{
			task: scheduledTask,
			kind: "",
		},
		{
			task: dispatchedTask,
			kind: AlreadyDispatched,
		},
		{
			task: cancelledTask,
			kind: AlreadyCancelled,
		},
		{
			task: doneTask,
			opt: ErrKindOption{
				SkipDispatchedAt: true,
			},
			kind: AlreadyDone,
		},
		{
			task: errTask,
			opt: ErrKindOption{
				SkipDispatchedAt: true,
			},
			kind: AlreadyDone,
		},
		{
			task: cancelledTask,
			opt: ErrKindOption{
				SkipCancelledAt: true,
			},
			kind: "",
		},
		{
			task: dispatchedTask,
			opt: ErrKindOption{
				SkipDispatchedAt: true,
			},
			kind: "",
		},
		{
			task: doneTask,
			opt: ErrKindOption{
				SkipDispatchedAt: true,
				SkipDoneAt:       true,
			},
			kind: "",
		},
		{
			task: dispatchedTask,
			opt: ErrKindOption{
				SkipDispatchedAt:          true,
				ReturnOnEmptyDispatchedAt: true,
			},
			kind: "",
		},
	} {
		assert.Equal(tc.kind, ErrKind(tc.task, tc.opt), "case = %+#v", tc)
	}
}

func TestErrKind_variants(t *testing.T) {
	assert := assert.New(t)

	type testCase struct {
		errFn     func(t Task) error
		assertFns [](func(error) bool)
	}
	for _, tc := range []testCase{
		{
			errFn: ErrKindUpdate,
			assertFns: [](func(error) bool){
				nil,
				IsAlreadyCancelled,
				IsAlreadyDispatched,
				IsAlreadyDone,
				IsAlreadyDone,
			},
		},
		{
			errFn: ErrKindCancel,
			assertFns: [](func(error) bool){
				nil,
				IsAlreadyCancelled,
				IsAlreadyDispatched,
				IsAlreadyDone,
				IsAlreadyDone,
			},
		},
		{
			errFn: ErrKindMarkAsDispatch,
			assertFns: [](func(error) bool){
				nil,
				IsAlreadyCancelled,
				IsAlreadyDispatched,
				IsAlreadyDone,
				IsAlreadyDone,
			},
		},
		{
			errFn: ErrKindMarkAsDone,
			assertFns: [](func(error) bool){
				IsNotDispatched,
				IsAlreadyCancelled,
				nil,
				IsAlreadyDone,
				IsAlreadyDone,
			},
		},
	} {
		for i := 0; i < len(eachStateTasks); i++ {
			err := tc.errFn(eachStateTasks[i])
			assertFn := tc.assertFns[i]
			if assertFn != nil {
				assert.True(assertFn(err))
			} else {
				assert.Nil(err)
			}
		}
	}
}
