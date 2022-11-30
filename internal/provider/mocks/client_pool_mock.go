// Code generated by MockGen. DO NOT EDIT.
// Source: internal/node/client_pool.go

// Package mock_client is a generated GoMock package.
package mock

// import (
// 	reflect "reflect"

// 	gomock "github.com/golang/mock/gomock"
// 	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
// 	subi "github.com/threefoldtech/terraform-provider-grid/pkg/subi"
// )

// // MockNodeClientCollection is a mock of NodeClientCollection interface.
// type MockNodeClientCollection struct {
// 	ctrl     *gomock.Controller
// 	recorder *MockNodeClientCollectionMockRecorder
// }

// // MockNodeClientCollectionMockRecorder is the mock recorder for MockNodeClientCollection.
// type MockNodeClientCollectionMockRecorder struct {
// 	mock *MockNodeClientCollection
// }

// // NewMockNodeClientCollection creates a new mock instance.
// func NewMockNodeClientCollection(ctrl *gomock.Controller) *MockNodeClientCollection {
// 	mock := &MockNodeClientCollection{ctrl: ctrl}
// 	mock.recorder = &MockNodeClientCollectionMockRecorder{mock}
// 	return mock
// }

// // EXPECT returns an object that allows the caller to indicate expected use.
// func (m *MockNodeClientCollection) EXPECT() *MockNodeClientCollectionMockRecorder {
// 	return m.recorder
// }

// // GetNodeClient mocks base method.
// func (m *MockNodeClientCollection) GetNodeClient(sub subi.SubstrateExt, nodeID uint32) (*client.NodeClient, error) {
// 	m.ctrl.T.Helper()
// 	ret := m.ctrl.Call(m, "GetNodeClient", sub, nodeID)
// 	ret0, _ := ret[0].(*client.NodeClient)
// 	ret1, _ := ret[1].(error)
// 	return ret0, ret1
// }

// // GetNodeClient indicates an expected call of GetNodeClient.
// func (mr *MockNodeClientCollectionMockRecorder) GetNodeClient(sub, nodeID interface{}) *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNodeClient", reflect.TypeOf((*MockNodeClientCollection)(nil).GetNodeClient), sub, nodeID)
// }
