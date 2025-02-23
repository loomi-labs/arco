// Code generated by MockGen. DO NOT EDIT.
// Source: backend/app/types/types.go
//
// Generated by this command:
//
//	mockgen -source=backend/app/types/types.go -destination=backend/app/mockapp/mocktypes/mocktypes.go --package=mocktypes
//

// Package mocktypes is a generated GoMock package.
package mocktypes

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockEventEmitter is a mock of EventEmitter interface.
type MockEventEmitter struct {
	ctrl     *gomock.Controller
	recorder *MockEventEmitterMockRecorder
	isgomock struct{}
}

// MockEventEmitterMockRecorder is the mock recorder for MockEventEmitter.
type MockEventEmitterMockRecorder struct {
	mock *MockEventEmitter
}

// NewMockEventEmitter creates a new mock instance.
func NewMockEventEmitter(ctrl *gomock.Controller) *MockEventEmitter {
	mock := &MockEventEmitter{ctrl: ctrl}
	mock.recorder = &MockEventEmitterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEventEmitter) EXPECT() *MockEventEmitterMockRecorder {
	return m.recorder
}

// EmitEvent mocks base method.
func (m *MockEventEmitter) EmitEvent(ctx context.Context, event string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "EmitEvent", ctx, event)
}

// EmitEvent indicates an expected call of EmitEvent.
func (mr *MockEventEmitterMockRecorder) EmitEvent(ctx, event any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EmitEvent", reflect.TypeOf((*MockEventEmitter)(nil).EmitEvent), ctx, event)
}
