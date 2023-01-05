// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/state/types.go

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	state "github.com/threefoldtech/terraform-provider-grid/pkg/state"
)

// MockDB is a mock of DB interface.
type MockDB struct {
	ctrl     *gomock.Controller
	recorder *MockDBMockRecorder
}

// MockDBMockRecorder is the mock recorder for MockDB.
type MockDBMockRecorder struct {
	mock *MockDB
}

// NewMockDB creates a new mock instance.
func NewMockDB(ctrl *gomock.Controller) *MockDB {
	mock := &MockDB{ctrl: ctrl}
	mock.recorder = &MockDBMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDB) EXPECT() *MockDBMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockDB) Delete() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete")
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockDBMockRecorder) Delete() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockDB)(nil).Delete))
}

// GetState mocks base method.
func (m *MockDB) GetState() state.State {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetState")
	ret0, _ := ret[0].(state.State)
	return ret0
}

// GetState indicates an expected call of GetState.
func (mr *MockDBMockRecorder) GetState() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetState", reflect.TypeOf((*MockDB)(nil).GetState))
}

// Load mocks base method.
func (m *MockDB) Load() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Load")
	ret0, _ := ret[0].(error)
	return ret0
}

// Load indicates an expected call of Load.
func (mr *MockDBMockRecorder) Load() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Load", reflect.TypeOf((*MockDB)(nil).Load))
}

// Save mocks base method.
func (m *MockDB) Save() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Save")
	ret0, _ := ret[0].(error)
	return ret0
}

// Save indicates an expected call of Save.
func (mr *MockDBMockRecorder) Save() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Save", reflect.TypeOf((*MockDB)(nil).Save))
}

// MockStateI is a mock of StateI interface.
type MockStateI struct {
	ctrl     *gomock.Controller
	recorder *MockStateIMockRecorder
}

// MockStateIMockRecorder is the mock recorder for MockStateI.
type MockStateIMockRecorder struct {
	mock *MockStateI
}

// NewMockStateI creates a new mock instance.
func NewMockStateI(ctrl *gomock.Controller) *MockStateI {
	mock := &MockStateI{ctrl: ctrl}
	mock.recorder = &MockStateIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStateI) EXPECT() *MockStateIMockRecorder {
	return m.recorder
}

// GetNetworkState mocks base method.
func (m *MockStateI) GetNetworkState() state.NetworkMap {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNetworkState")
	ret0, _ := ret[0].(state.NetworkMap)
	return ret0
}

// GetNetworkState indicates an expected call of GetNetworkState.
func (mr *MockStateIMockRecorder) GetNetworkState() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNetworkState", reflect.TypeOf((*MockStateI)(nil).GetNetworkState))
}

// Marshal mocks base method.
func (m *MockStateI) Marshal() ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Marshal")
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Marshal indicates an expected call of Marshal.
func (mr *MockStateIMockRecorder) Marshal() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Marshal", reflect.TypeOf((*MockStateI)(nil).Marshal))
}

// Unmarshal mocks base method.
func (m *MockStateI) Unmarshal(data []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Unmarshal", data)
	ret0, _ := ret[0].(error)
	return ret0
}

// Unmarshal indicates an expected call of Unmarshal.
func (mr *MockStateIMockRecorder) Unmarshal(data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unmarshal", reflect.TypeOf((*MockStateI)(nil).Unmarshal), data)
}

// MockNetworkState is a mock of NetworkState interface.
type MockNetworkState struct {
	ctrl     *gomock.Controller
	recorder *MockNetworkStateMockRecorder
}

// MockNetworkStateMockRecorder is the mock recorder for MockNetworkState.
type MockNetworkStateMockRecorder struct {
	mock *MockNetworkState
}

// NewMockNetworkState creates a new mock instance.
func NewMockNetworkState(ctrl *gomock.Controller) *MockNetworkState {
	mock := &MockNetworkState{ctrl: ctrl}
	mock.recorder = &MockNetworkStateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNetworkState) EXPECT() *MockNetworkStateMockRecorder {
	return m.recorder
}

// DeleteNetwork mocks base method.
func (m *MockNetworkState) DeleteNetwork(networkName string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "DeleteNetwork", networkName)
}

// DeleteNetwork indicates an expected call of DeleteNetwork.
func (mr *MockNetworkStateMockRecorder) DeleteNetwork(networkName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNetwork", reflect.TypeOf((*MockNetworkState)(nil).DeleteNetwork), networkName)
}

// GetNetwork mocks base method.
func (m *MockNetworkState) GetNetwork(networkName string) state.NetworkInterface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNetwork", networkName)
	ret0, _ := ret[0].(state.NetworkInterface)
	return ret0
}

// GetNetwork indicates an expected call of GetNetwork.
func (mr *MockNetworkStateMockRecorder) GetNetwork(networkName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNetwork", reflect.TypeOf((*MockNetworkState)(nil).GetNetwork), networkName)
}

// MockNetwork is a mock of Network interface.
type MockNetwork struct {
	ctrl     *gomock.Controller
	recorder *MockNetworkMockRecorder
}

// MockNetworkMockRecorder is the mock recorder for MockNetwork.
type MockNetworkMockRecorder struct {
	mock *MockNetwork
}

// NewMockNetwork creates a new mock instance.
func NewMockNetwork(ctrl *gomock.Controller) *MockNetwork {
	mock := &MockNetwork{ctrl: ctrl}
	mock.recorder = &MockNetworkMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNetwork) EXPECT() *MockNetworkMockRecorder {
	return m.recorder
}

// DeleteDeployment mocks base method.
func (m *MockNetwork) DeleteDeployment(nodeID uint32, deploymentID string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "DeleteDeployment", nodeID, deploymentID)
}

// DeleteDeployment indicates an expected call of DeleteDeployment.
func (mr *MockNetworkMockRecorder) DeleteDeployment(nodeID, deploymentID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDeployment", reflect.TypeOf((*MockNetwork)(nil).DeleteDeployment), nodeID, deploymentID)
}

// DeleteNodeSubnet mocks base method.
func (m *MockNetwork) DeleteNodeSubnet(nodeID uint32) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "DeleteNodeSubnet", nodeID)
}

// DeleteNodeSubnet indicates an expected call of DeleteNodeSubnet.
func (mr *MockNetworkMockRecorder) DeleteNodeSubnet(nodeID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNodeSubnet", reflect.TypeOf((*MockNetwork)(nil).DeleteNodeSubnet), nodeID)
}

// GetDeploymentHostIDs mocks base method.
func (m *MockNetwork) GetDeploymentHostIDs(nodeID uint32, deploymentID string) []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDeploymentHostIDs", nodeID, deploymentID)
	ret0, _ := ret[0].([]byte)
	return ret0
}

// GetDeploymentHostIDs indicates an expected call of GetDeploymentHostIDs.
func (mr *MockNetworkMockRecorder) GetDeploymentHostIDs(nodeID, deploymentID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDeploymentHostIDs", reflect.TypeOf((*MockNetwork)(nil).GetDeploymentHostIDs), nodeID, deploymentID)
}

// GetNodeDeploymentHostIDs mocks base method.
func (m *MockNetwork) GetNodeDeploymentHostIDs() state.NodeDeploymentHostIDs {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNodeDeploymentHostIDs")
	ret0, _ := ret[0].(state.NodeDeploymentHostIDs)
	return ret0
}

// GetNodeDeploymentHostIDs indicates an expected call of GetNodeDeploymentHostIDs.
func (mr *MockNetworkMockRecorder) GetNodeDeploymentHostIDs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNodeDeploymentHostIDs", reflect.TypeOf((*MockNetwork)(nil).GetNodeDeploymentHostIDs))
}

// GetUsedNetworkHostIDs mocks base method.
func (m *MockNetwork) GetUsedNetworkHostIDs(nodeID uint32) []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUsedNetworkHostIDs", nodeID)
	ret0, _ := ret[0].([]byte)
	return ret0
}

// GetUsedNetworkHostIDs indicates an expected call of GetUsedNetworkHostIDs.
func (mr *MockNetworkMockRecorder) GetUsedNetworkHostIDs(nodeID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUsedNetworkHostIDs", reflect.TypeOf((*MockNetwork)(nil).GetUsedNetworkHostIDs), nodeID)
}

// GetNodeSubnet mocks base method.
func (m *MockNetwork) GetNodeSubnet(nodeID uint32) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNodeSubnet", nodeID)
	ret0, _ := ret[0].(string)
	return ret0
}

// GetNodeSubnet indicates an expected call of GetNodeSubnet.
func (mr *MockNetworkMockRecorder) GetNodeSubnet(nodeID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNodeSubnet", reflect.TypeOf((*MockNetwork)(nil).GetNodeSubnet), nodeID)
}

// GetSubnets mocks base method.
func (m *MockNetwork) GetSubnets() map[uint32]string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSubnets")
	ret0, _ := ret[0].(map[uint32]string)
	return ret0
}

// GetSubnets indicates an expected call of GetSubnets.
func (mr *MockNetworkMockRecorder) GetSubnets() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSubnets", reflect.TypeOf((*MockNetwork)(nil).GetSubnets))
}

// SetDeploymentHostIDs mocks base method.
func (m *MockNetwork) SetDeploymentHostIDs(nodeID uint32, deploymentID string, ips []byte) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetDeploymentHostIDs", nodeID, deploymentID, ips)
}

// SetDeploymentHostIDs indicates an expected call of SetDeploymentHostIDs.
func (mr *MockNetworkMockRecorder) SetDeploymentHostIDs(nodeID, deploymentID, ips interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetDeploymentHostIDs", reflect.TypeOf((*MockNetwork)(nil).SetDeploymentHostIDs), nodeID, deploymentID, ips)
}

// SetNodeSubnet mocks base method.
func (m *MockNetwork) SetNodeSubnet(nodeID uint32, subnet string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetNodeSubnet", nodeID, subnet)
}

// SetNodeSubnet indicates an expected call of SetNodeSubnet.
func (mr *MockNetworkMockRecorder) SetNodeSubnet(nodeID, subnet interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetNodeSubnet", reflect.TypeOf((*MockNetwork)(nil).SetNodeSubnet), nodeID, subnet)
}
