// Code generated by MockGen. DO NOT EDIT.
// Source: backend/borg/borg.go
//
// Generated by this command:
//
//	mockgen -source=backend/borg/borg.go -destination=backend/borg/mockborg/mockborg.go --package=mockborg
//

// Package mockborg is a generated GoMock package.
package mockborg

import (
	borg "arco/backend/borg"
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockBorg is a mock of Borg interface.
type MockBorg struct {
	ctrl     *gomock.Controller
	recorder *MockBorgMockRecorder
}

// MockBorgMockRecorder is the mock recorder for MockBorg.
type MockBorgMockRecorder struct {
	mock *MockBorg
}

// NewMockBorg creates a new mock instance.
func NewMockBorg(ctrl *gomock.Controller) *MockBorg {
	mock := &MockBorg{ctrl: ctrl}
	mock.recorder = &MockBorgMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBorg) EXPECT() *MockBorgMockRecorder {
	return m.recorder
}

// BreakLock mocks base method.
func (m *MockBorg) BreakLock(ctx context.Context, repository, password string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BreakLock", ctx, repository, password)
	ret0, _ := ret[0].(error)
	return ret0
}

// BreakLock indicates an expected call of BreakLock.
func (mr *MockBorgMockRecorder) BreakLock(ctx, repository, password any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BreakLock", reflect.TypeOf((*MockBorg)(nil).BreakLock), ctx, repository, password)
}

// Compact mocks base method.
func (m *MockBorg) Compact(ctx context.Context, repoUrl, repoPassword string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Compact", ctx, repoUrl, repoPassword)
	ret0, _ := ret[0].(error)
	return ret0
}

// Compact indicates an expected call of Compact.
func (mr *MockBorgMockRecorder) Compact(ctx, repoUrl, repoPassword any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Compact", reflect.TypeOf((*MockBorg)(nil).Compact), ctx, repoUrl, repoPassword)
}

// Create mocks base method.
func (m *MockBorg) Create(ctx context.Context, repoUrl, password, prefix string, backupPaths, excludePaths []string, ch chan borg.BackupProgress) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, repoUrl, password, prefix, backupPaths, excludePaths, ch)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockBorgMockRecorder) Create(ctx, repoUrl, password, prefix, backupPaths, excludePaths, ch any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockBorg)(nil).Create), ctx, repoUrl, password, prefix, backupPaths, excludePaths, ch)
}

// DeleteArchive mocks base method.
func (m *MockBorg) DeleteArchive(ctx context.Context, repository, archive, password string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteArchive", ctx, repository, archive, password)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteArchive indicates an expected call of DeleteArchive.
func (mr *MockBorgMockRecorder) DeleteArchive(ctx, repository, archive, password any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteArchive", reflect.TypeOf((*MockBorg)(nil).DeleteArchive), ctx, repository, archive, password)
}

// DeleteArchives mocks base method.
func (m *MockBorg) DeleteArchives(ctx context.Context, repoUrl, password, prefix string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteArchives", ctx, repoUrl, password, prefix)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteArchives indicates an expected call of DeleteArchives.
func (mr *MockBorgMockRecorder) DeleteArchives(ctx, repoUrl, password, prefix any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteArchives", reflect.TypeOf((*MockBorg)(nil).DeleteArchives), ctx, repoUrl, password, prefix)
}

// Info mocks base method.
func (m *MockBorg) Info(url, password string) (*borg.InfoResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Info", url, password)
	ret0, _ := ret[0].(*borg.InfoResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Info indicates an expected call of Info.
func (mr *MockBorgMockRecorder) Info(url, password any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Info", reflect.TypeOf((*MockBorg)(nil).Info), url, password)
}

// Init mocks base method.
func (m *MockBorg) Init(url, password string, noPassword bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Init", url, password, noPassword)
	ret0, _ := ret[0].(error)
	return ret0
}

// Init indicates an expected call of Init.
func (mr *MockBorgMockRecorder) Init(url, password, noPassword any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Init", reflect.TypeOf((*MockBorg)(nil).Init), url, password, noPassword)
}

// List mocks base method.
func (m *MockBorg) List(repoUrl, password string) (*borg.ListResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", repoUrl, password)
	ret0, _ := ret[0].(*borg.ListResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockBorgMockRecorder) List(repoUrl, password any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockBorg)(nil).List), repoUrl, password)
}

// MountArchive mocks base method.
func (m *MockBorg) MountArchive(repoUrl, archive, password, mountPath string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MountArchive", repoUrl, archive, password, mountPath)
	ret0, _ := ret[0].(error)
	return ret0
}

// MountArchive indicates an expected call of MountArchive.
func (mr *MockBorgMockRecorder) MountArchive(repoUrl, archive, password, mountPath any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MountArchive", reflect.TypeOf((*MockBorg)(nil).MountArchive), repoUrl, archive, password, mountPath)
}

// MountRepository mocks base method.
func (m *MockBorg) MountRepository(repoUrl, password, mountPath string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MountRepository", repoUrl, password, mountPath)
	ret0, _ := ret[0].(error)
	return ret0
}

// MountRepository indicates an expected call of MountRepository.
func (mr *MockBorgMockRecorder) MountRepository(repoUrl, password, mountPath any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MountRepository", reflect.TypeOf((*MockBorg)(nil).MountRepository), repoUrl, password, mountPath)
}

// Prune mocks base method.
func (m *MockBorg) Prune(ctx context.Context, repoUrl, password, prefix string, isDryRun bool, ch chan borg.PruneResult) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Prune", ctx, repoUrl, password, prefix, isDryRun, ch)
	ret0, _ := ret[0].(error)
	return ret0
}

// Prune indicates an expected call of Prune.
func (mr *MockBorgMockRecorder) Prune(ctx, repoUrl, password, prefix, isDryRun, ch any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Prune", reflect.TypeOf((*MockBorg)(nil).Prune), ctx, repoUrl, password, prefix, isDryRun, ch)
}

// Umount mocks base method.
func (m *MockBorg) Umount(path string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Umount", path)
	ret0, _ := ret[0].(error)
	return ret0
}

// Umount indicates an expected call of Umount.
func (mr *MockBorgMockRecorder) Umount(path any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Umount", reflect.TypeOf((*MockBorg)(nil).Umount), path)
}