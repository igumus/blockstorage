// Code generated by MockGen. DO NOT EDIT.
// Source: peer.go

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	objectstore "github.com/igumus/go-objectstore-lib"
	cid "github.com/ipfs/go-cid"
)

// MockBlockStoragePeer is a mock of BlockStoragePeer interface.
type MockBlockStoragePeer struct {
	ctrl     *gomock.Controller
	recorder *MockBlockStoragePeerMockRecorder
}

// MockBlockStoragePeerMockRecorder is the mock recorder for MockBlockStoragePeer.
type MockBlockStoragePeerMockRecorder struct {
	mock *MockBlockStoragePeer
}

// NewMockBlockStoragePeer creates a new mock instance.
func NewMockBlockStoragePeer(ctrl *gomock.Controller) *MockBlockStoragePeer {
	mock := &MockBlockStoragePeer{ctrl: ctrl}
	mock.recorder = &MockBlockStoragePeerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBlockStoragePeer) EXPECT() *MockBlockStoragePeerMockRecorder {
	return m.recorder
}

// AnnounceBlock mocks base method.
func (m *MockBlockStoragePeer) AnnounceBlock(arg0 context.Context, arg1 cid.Cid) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AnnounceBlock", arg0, arg1)
	ret0, _ := ret[0].(bool)
	return ret0
}

// AnnounceBlock indicates an expected call of AnnounceBlock.
func (mr *MockBlockStoragePeerMockRecorder) AnnounceBlock(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AnnounceBlock", reflect.TypeOf((*MockBlockStoragePeer)(nil).AnnounceBlock), arg0, arg1)
}

// GetRemoteBlock mocks base method.
func (m *MockBlockStoragePeer) GetRemoteBlock(arg0 context.Context, arg1 cid.Cid) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRemoteBlock", arg0, arg1)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRemoteBlock indicates an expected call of GetRemoteBlock.
func (mr *MockBlockStoragePeerMockRecorder) GetRemoteBlock(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRemoteBlock", reflect.TypeOf((*MockBlockStoragePeer)(nil).GetRemoteBlock), arg0, arg1)
}

// RegisterReadProtocol mocks base method.
func (m *MockBlockStoragePeer) RegisterReadProtocol(arg0 context.Context, arg1 objectstore.ObjectStore) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RegisterReadProtocol", arg0, arg1)
}

// RegisterReadProtocol indicates an expected call of RegisterReadProtocol.
func (mr *MockBlockStoragePeerMockRecorder) RegisterReadProtocol(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterReadProtocol", reflect.TypeOf((*MockBlockStoragePeer)(nil).RegisterReadProtocol), arg0, arg1)
}
