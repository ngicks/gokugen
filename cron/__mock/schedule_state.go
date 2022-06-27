// Code generated by MockGen. DO NOT EDIT.
// Source: schedule_state.go

// Package mock_cron is a generated GoMock package.
package mock_cron

import (
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
)

// MockNextScheduler is a mock of NextScheduler interface.
type MockNextScheduler struct {
	ctrl     *gomock.Controller
	recorder *MockNextSchedulerMockRecorder
}

// MockNextSchedulerMockRecorder is the mock recorder for MockNextScheduler.
type MockNextSchedulerMockRecorder struct {
	mock *MockNextScheduler
}

// NewMockNextScheduler creates a new mock instance.
func NewMockNextScheduler(ctrl *gomock.Controller) *MockNextScheduler {
	mock := &MockNextScheduler{ctrl: ctrl}
	mock.recorder = &MockNextSchedulerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNextScheduler) EXPECT() *MockNextSchedulerMockRecorder {
	return m.recorder
}

// NextSchedule mocks base method.
func (m *MockNextScheduler) NextSchedule(now time.Time) (time.Time, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextSchedule", now)
	ret0, _ := ret[0].(time.Time)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NextSchedule indicates an expected call of NextSchedule.
func (mr *MockNextSchedulerMockRecorder) NextSchedule(now interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextSchedule", reflect.TypeOf((*MockNextScheduler)(nil).NextSchedule), now)
}
