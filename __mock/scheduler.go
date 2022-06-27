// Code generated by MockGen. DO NOT EDIT.
// Source: scheduler.go

// Package mock_gokugen is a generated GoMock package.
package mock_gokugen

import (
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
	gokugen "github.com/ngicks/gokugen"
	scheduler "github.com/ngicks/gokugen/scheduler"
)

// MockScheduler is a mock of Scheduler interface.
type MockScheduler struct {
	ctrl     *gomock.Controller
	recorder *MockSchedulerMockRecorder
}

// MockSchedulerMockRecorder is the mock recorder for MockScheduler.
type MockSchedulerMockRecorder struct {
	mock *MockScheduler
}

// NewMockScheduler creates a new mock instance.
func NewMockScheduler(ctrl *gomock.Controller) *MockScheduler {
	mock := &MockScheduler{ctrl: ctrl}
	mock.recorder = &MockSchedulerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockScheduler) EXPECT() *MockSchedulerMockRecorder {
	return m.recorder
}

// Schedule mocks base method.
func (m *MockScheduler) Schedule(task *scheduler.Task) (*scheduler.TaskController, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Schedule", task)
	ret0, _ := ret[0].(*scheduler.TaskController)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Schedule indicates an expected call of Schedule.
func (mr *MockSchedulerMockRecorder) Schedule(task interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Schedule", reflect.TypeOf((*MockScheduler)(nil).Schedule), task)
}

// MockTask is a mock of Task interface.
type MockTask struct {
	ctrl     *gomock.Controller
	recorder *MockTaskMockRecorder
}

// MockTaskMockRecorder is the mock recorder for MockTask.
type MockTaskMockRecorder struct {
	mock *MockTask
}

// NewMockTask creates a new mock instance.
func NewMockTask(ctrl *gomock.Controller) *MockTask {
	mock := &MockTask{ctrl: ctrl}
	mock.recorder = &MockTaskMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTask) EXPECT() *MockTaskMockRecorder {
	return m.recorder
}

// Cancel mocks base method.
func (m *MockTask) Cancel() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cancel")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Cancel indicates an expected call of Cancel.
func (mr *MockTaskMockRecorder) Cancel() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cancel", reflect.TypeOf((*MockTask)(nil).Cancel))
}

// CancelWithReason mocks base method.
func (m *MockTask) CancelWithReason(err error) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CancelWithReason", err)
	ret0, _ := ret[0].(bool)
	return ret0
}

// CancelWithReason indicates an expected call of CancelWithReason.
func (mr *MockTaskMockRecorder) CancelWithReason(err interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CancelWithReason", reflect.TypeOf((*MockTask)(nil).CancelWithReason), err)
}

// GetScheduledTime mocks base method.
func (m *MockTask) GetScheduledTime() time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetScheduledTime")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// GetScheduledTime indicates an expected call of GetScheduledTime.
func (mr *MockTaskMockRecorder) GetScheduledTime() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetScheduledTime", reflect.TypeOf((*MockTask)(nil).GetScheduledTime))
}

// IsCancelled mocks base method.
func (m *MockTask) IsCancelled() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsCancelled")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsCancelled indicates an expected call of IsCancelled.
func (mr *MockTaskMockRecorder) IsCancelled() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsCancelled", reflect.TypeOf((*MockTask)(nil).IsCancelled))
}

// IsDone mocks base method.
func (m *MockTask) IsDone() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsDone")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsDone indicates an expected call of IsDone.
func (mr *MockTaskMockRecorder) IsDone() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsDone", reflect.TypeOf((*MockTask)(nil).IsDone))
}

// MockSchedulerContext is a mock of SchedulerContext interface.
type MockSchedulerContext struct {
	ctrl     *gomock.Controller
	recorder *MockSchedulerContextMockRecorder
}

// MockSchedulerContextMockRecorder is the mock recorder for MockSchedulerContext.
type MockSchedulerContextMockRecorder struct {
	mock *MockSchedulerContext
}

// NewMockSchedulerContext creates a new mock instance.
func NewMockSchedulerContext(ctrl *gomock.Controller) *MockSchedulerContext {
	mock := &MockSchedulerContext{ctrl: ctrl}
	mock.recorder = &MockSchedulerContextMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSchedulerContext) EXPECT() *MockSchedulerContextMockRecorder {
	return m.recorder
}

// ScheduledTime mocks base method.
func (m *MockSchedulerContext) ScheduledTime() time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ScheduledTime")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// ScheduledTime indicates an expected call of ScheduledTime.
func (mr *MockSchedulerContextMockRecorder) ScheduledTime() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ScheduledTime", reflect.TypeOf((*MockSchedulerContext)(nil).ScheduledTime))
}

// Value mocks base method.
func (m *MockSchedulerContext) Value(key any) (any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Value", key)
	ret0, _ := ret[0].(any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Value indicates an expected call of Value.
func (mr *MockSchedulerContextMockRecorder) Value(key interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Value", reflect.TypeOf((*MockSchedulerContext)(nil).Value), key)
}

// Work mocks base method.
func (m *MockSchedulerContext) Work() gokugen.WorkFn {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Work")
	ret0, _ := ret[0].(gokugen.WorkFn)
	return ret0
}

// Work indicates an expected call of Work.
func (mr *MockSchedulerContextMockRecorder) Work() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Work", reflect.TypeOf((*MockSchedulerContext)(nil).Work))
}