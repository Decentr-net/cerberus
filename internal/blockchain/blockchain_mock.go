// Code generated by MockGen. DO NOT EDIT.
// Source: blockchain.go

// Package blockchain is a generated GoMock package.
package blockchain

import (
	context "context"
	types "github.com/cosmos/cosmos-sdk/types"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockBlockchain is a mock of Blockchain interface
type MockBlockchain struct {
	ctrl     *gomock.Controller
	recorder *MockBlockchainMockRecorder
}

// MockBlockchainMockRecorder is the mock recorder for MockBlockchain
type MockBlockchainMockRecorder struct {
	mock *MockBlockchain
}

// NewMockBlockchain creates a new mock instance
func NewMockBlockchain(ctrl *gomock.Controller) *MockBlockchain {
	mock := &MockBlockchain{ctrl: ctrl}
	mock.recorder = &MockBlockchainMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockBlockchain) EXPECT() *MockBlockchainMockRecorder {
	return m.recorder
}

// Ping mocks base method
func (m *MockBlockchain) Ping(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Ping", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Ping indicates an expected call of Ping
func (mr *MockBlockchainMockRecorder) Ping(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Ping", reflect.TypeOf((*MockBlockchain)(nil).Ping), ctx)
}

// DistributeReward mocks base method
func (m *MockBlockchain) DistributeReward(receiver types.AccAddress, id, reward uint64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DistributeReward", receiver, id, reward)
	ret0, _ := ret[0].(error)
	return ret0
}

// DistributeReward indicates an expected call of DistributeReward
func (mr *MockBlockchainMockRecorder) DistributeReward(receiver, id, reward interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DistributeReward", reflect.TypeOf((*MockBlockchain)(nil).DistributeReward), receiver, id, reward)
}