// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/cosmos/cosmos-sdk/x/testmodule (interfaces: BankKeeper)

// Package mock_testmodule is a generated GoMock package.
package testmodule

import (
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockBankKeeper is a mock of BankKeeper interface
type MockBankKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockBankKeeperMockRecorder
}

// MockBankKeeperMockRecorder is the mock recorder for MockBankKeeper
type MockBankKeeperMockRecorder struct {
	mock *MockBankKeeper
}

// NewMockBankKeeper creates a new mock instance
func NewMockBankKeeper(ctrl *gomock.Controller) *MockBankKeeper {
	mock := &MockBankKeeper{ctrl: ctrl}
	mock.recorder = &MockBankKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockBankKeeper) EXPECT() *MockBankKeeperMockRecorder {
	return m.recorder
}

// TestFunc mocks base method
func (m *MockBankKeeper) TestFunc(arg0 int) int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TestFunc", arg0)
	ret0, _ := ret[0].(int)
	return ret0
}

// TestFunc indicates an expected call of TestFunc
func (mr *MockBankKeeperMockRecorder) TestFunc(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TestFunc", reflect.TypeOf((*MockBankKeeper)(nil).TestFunc), arg0)
}