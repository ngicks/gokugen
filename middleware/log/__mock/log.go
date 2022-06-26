// Code generated by MockGen. DO NOT EDIT.
// Source: log.go

// Package mock_log is a generated GoMock package.
package mock_log

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockLogger is a mock of Logger interface.
type MockLogger struct {
	ctrl     *gomock.Controller
	recorder *MockLoggerMockRecorder
}

// MockLoggerMockRecorder is the mock recorder for MockLogger.
type MockLoggerMockRecorder struct {
	mock *MockLogger
}

// NewMockLogger creates a new mock instance.
func NewMockLogger(ctrl *gomock.Controller) *MockLogger {
	mock := &MockLogger{ctrl: ctrl}
	mock.recorder = &MockLoggerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLogger) EXPECT() *MockLoggerMockRecorder {
	return m.recorder
}

// Error mocks base method.
func (m *MockLogger) Error(taskId, workId string, e error) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Error", taskId, workId, e)
}

// Error indicates an expected call of Error.
func (mr *MockLoggerMockRecorder) Error(taskId, workId, e interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Error", reflect.TypeOf((*MockLogger)(nil).Error), taskId, workId, e)
}

// Info mocks base method.
func (m *MockLogger) Info(taskId, workId string, v any) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Info", taskId, workId, v)
}

// Info indicates an expected call of Info.
func (mr *MockLoggerMockRecorder) Info(taskId, workId, v interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Info", reflect.TypeOf((*MockLogger)(nil).Info), taskId, workId, v)
}
