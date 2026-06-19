// Package mocks contains a gomock-compatible mock of store.Store for handler
// and dispatch tests. It is hand-written (rather than mockgen-generated) so the
// project does not need to take a build-time dependency on golang.org/x/tools.
// It implements the same surface mockgen would produce.
package mocks

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"

	store "github.com/vsjwl/mm-time-tracking/server/store"
)

// MockStore is a mock of the store.Store interface.
type MockStore struct {
	ctrl     *gomock.Controller
	recorder *MockStoreMockRecorder
}

// MockStoreMockRecorder is the mock recorder for MockStore.
type MockStoreMockRecorder struct {
	mock *MockStore
}

// NewMockStore creates a new mock instance.
func NewMockStore(ctrl *gomock.Controller) *MockStore {
	mock := &MockStore{ctrl: ctrl}
	mock.recorder = &MockStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStore) EXPECT() *MockStoreMockRecorder {
	return m.recorder
}

// Compile-time assertion that MockStore satisfies store.Store.
var _ store.Store = (*MockStore)(nil)

// StartTimer mocks base method.
func (m *MockStore) StartTimer(userID, project, description string, now int64) (*store.TimeEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StartTimer", userID, project, description, now)
	ret0, _ := ret[0].(*store.TimeEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StartTimer indicates an expected call of StartTimer.
func (mr *MockStoreMockRecorder) StartTimer(userID, project, description, now interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StartTimer", reflect.TypeOf((*MockStore)(nil).StartTimer), userID, project, description, now)
}

// StopTimer mocks base method.
func (m *MockStore) StopTimer(userID string, now int64) (*store.TimeEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StopTimer", userID, now)
	ret0, _ := ret[0].(*store.TimeEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StopTimer indicates an expected call of StopTimer.
func (mr *MockStoreMockRecorder) StopTimer(userID, now interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StopTimer", reflect.TypeOf((*MockStore)(nil).StopTimer), userID, now)
}

// StartBreak mocks base method.
func (m *MockStore) StartBreak(userID string, now int64) (*store.TimeEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StartBreak", userID, now)
	ret0, _ := ret[0].(*store.TimeEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StartBreak indicates an expected call of StartBreak.
func (mr *MockStoreMockRecorder) StartBreak(userID, now interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StartBreak", reflect.TypeOf((*MockStore)(nil).StartBreak), userID, now)
}

// StopBreak mocks base method.
func (m *MockStore) StopBreak(userID string, now int64) (*store.TimeEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StopBreak", userID, now)
	ret0, _ := ret[0].(*store.TimeEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StopBreak indicates an expected call of StopBreak.
func (mr *MockStoreMockRecorder) StopBreak(userID, now interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StopBreak", reflect.TypeOf((*MockStore)(nil).StopBreak), userID, now)
}

// GetRunning mocks base method.
func (m *MockStore) GetRunning(userID string) (*store.TimeEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRunning", userID)
	ret0, _ := ret[0].(*store.TimeEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRunning indicates an expected call of GetRunning.
func (mr *MockStoreMockRecorder) GetRunning(userID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRunning", reflect.TypeOf((*MockStore)(nil).GetRunning), userID)
}

// Get mocks base method.
func (m *MockStore) Get(id string) (*store.TimeEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", id)
	ret0, _ := ret[0].(*store.TimeEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockStoreMockRecorder) Get(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockStore)(nil).Get), id)
}

// List mocks base method.
func (m *MockStore) List(userID string, from, to int64) ([]*store.TimeEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", userID, from, to)
	ret0, _ := ret[0].([]*store.TimeEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockStoreMockRecorder) List(userID, from, to interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockStore)(nil).List), userID, from, to)
}

// ListAll mocks base method.
func (m *MockStore) ListAll(userID string, from, to int64) ([]*store.TimeEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAll", userID, from, to)
	ret0, _ := ret[0].([]*store.TimeEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAll indicates an expected call of ListAll.
func (mr *MockStoreMockRecorder) ListAll(userID, from, to interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAll", reflect.TypeOf((*MockStore)(nil).ListAll), userID, from, to)
}

// Create mocks base method.
func (m *MockStore) Create(e *store.TimeEntry) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", e)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockStoreMockRecorder) Create(e interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockStore)(nil).Create), e)
}

// Update mocks base method.
func (m *MockStore) Update(e *store.TimeEntry) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", e)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockStoreMockRecorder) Update(e interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockStore)(nil).Update), e)
}

// Delete mocks base method.
func (m *MockStore) Delete(userID, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", userID, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockStoreMockRecorder) Delete(userID, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockStore)(nil).Delete), userID, id)
}

// SetStatusRange mocks base method.
func (m *MockStore) SetStatusRange(userID string, from, to int64, fromStatus, toStatus string) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetStatusRange", userID, from, to, fromStatus, toStatus)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SetStatusRange indicates an expected call of SetStatusRange.
func (mr *MockStoreMockRecorder) SetStatusRange(userID, from, to, fromStatus, toStatus interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetStatusRange", reflect.TypeOf((*MockStore)(nil).SetStatusRange), userID, from, to, fromStatus, toStatus)
}

// Suggestions mocks base method.
func (m *MockStore) Suggestions(userID string, limit int) ([]string, []string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Suggestions", userID, limit)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].([]string)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Suggestions indicates an expected call of Suggestions.
func (mr *MockStoreMockRecorder) Suggestions(userID, limit interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Suggestions", reflect.TypeOf((*MockStore)(nil).Suggestions), userID, limit)
}
