package provider

import (
	"context"
	"math/big"
	"testing"

	types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/substrate-client"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/terraform-provider-grid/pkg/deployer"
	"github.com/threefoldtech/terraform-provider-grid/pkg/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func TestNameValidateBadAccount(t *testing.T) {
	ctrl := gomock.NewController(t)

	// Assert that Bar() is invoked.
	defer ctrl.Finish()

	m := NewMockSubstrateClient(ctrl)

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	m.
		EXPECT().
		GetAccount(gomock.Eq(identity)).
		Return(types.AccountInfo{}, errors.New("bad account"))
	gw := GatewayNameDeployer{
		APIClient: &apiClient{
			identity: identity,
		},
	}
	err = gw.Validate(context.TODO(), m)
	assert.Error(t, err)
}

func TestNameValidateEnoughMoneyNodeNotReachable(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	sub := NewMockSubstrateClient(ctrl)
	cl := NewMockClient(ctrl)
	pool := NewMockNodeClientCollection(ctrl)
	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	sub.
		EXPECT().
		GetAccount(gomock.Eq(identity)).
		Return(types.AccountInfo{
			Data: struct {
				Free       types.U128
				Reserved   types.U128
				MiscFrozen types.U128
				FreeFrozen types.U128
			}{
				Free: types.NewU128(*big.NewInt(30000)),
			},
		}, nil)
	cl.
		EXPECT().
		Call(
			gomock.Any(),
			uint32(10),
			"zos.network.interfaces",
			nil,
			gomock.Any(),
		).
		Return(errors.New("couldn't reach node"))
	pool.
		EXPECT().
		GetNodeClient(
			gomock.Any(),
			uint32(11),
		).
		Return(client.NewNodeClient(10, cl), nil)

	gw := GatewayNameDeployer{
		APIClient: &apiClient{
			identity: identity,
		},
		ncPool: pool,
		Node:   11,
	}
	err = gw.Validate(context.TODO(), sub)
	assert.Error(t, err)
}

func TestNameValidateEnoughMoneyNodeReachable(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	sub := NewMockSubstrateClient(ctrl)
	cl := NewMockClient(ctrl)
	pool := NewMockNodeClientCollection(ctrl)
	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	sub.
		EXPECT().
		GetAccount(gomock.Eq(identity)).
		Return(types.AccountInfo{
			Data: struct {
				Free       types.U128
				Reserved   types.U128
				MiscFrozen types.U128
				FreeFrozen types.U128
			}{
				Free: types.NewU128(*big.NewInt(30000)),
			},
		}, nil)
	cl.
		EXPECT().
		Call(
			gomock.Any(),
			uint32(10),
			"zos.network.interfaces",
			nil,
			gomock.Any(),
		).
		Return(nil)
	pool.
		EXPECT().
		GetNodeClient(
			gomock.Any(),
			uint32(11),
		).
		Return(client.NewNodeClient(10, cl), nil)

	gw := GatewayNameDeployer{
		APIClient: &apiClient{
			identity: identity,
		},
		ncPool: pool,
		Node:   11,
	}
	err = gw.Validate(context.TODO(), sub)
	assert.NoError(t, err)
}

func TestNameGenerateDeployment(t *testing.T) {
	g := workloads.GatewayNameProxy{
		Name:           "name",
		TLSPassthrough: false,
		Backends:       []zos.Backend{"a", "b"},
		FQDN:           "name.com",
	}
	gw := GatewayNameDeployer{
		APIClient: &apiClient{
			twin_id: 11,
		},
		Node: 10,
		Gw:   g,
	}
	dls, err := gw.GenerateVersionlessDeployments(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, dls, map[uint32]gridtypes.Deployment{
		10: {
			Version: 0,
			TwinID:  11,
			Workloads: []gridtypes.Workload{
				{
					Version: 0,
					Type:    zos.GatewayNameProxyType,
					Name:    gridtypes.Name(g.Name),
					Data: gridtypes.MustMarshal(zos.GatewayNameProxy{
						TLSPassthrough: g.TLSPassthrough,
						Backends:       g.Backends,
						Name:           g.Name,
					}),
				},
			},
			SignatureRequirement: gridtypes.SignatureRequirement{
				WeightRequired: 1,
				Requests: []gridtypes.SignatureRequest{
					{
						TwinID: 11,
						Weight: 1,
					},
				},
			},
		},
	})
}

func TestNameDeploy(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	deployer := deployer.NewMockDeployer(ctrl)
	sub := NewMockSubstrateClient(ctrl)
	gw := GatewayNameDeployer{
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		Node: 10,
		Gw: workloads.GatewayNameProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		deployer: deployer,
	}
	dls, err := gw.GenerateVersionlessDeployments(context.Background())
	assert.NoError(t, err)
	deployer.EXPECT().Deploy(
		gomock.Any(),
		sub,
		nil,
		dls,
	).Return(map[uint32]uint64{10: 100}, nil)
	sub.EXPECT().
		CreateNameContract(identity, "name").
		Return(uint64(100), nil)
	err = gw.Deploy(context.Background(), sub)
	assert.NoError(t, err)
	assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{10: 100})
}

func TestNameUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	deployer := deployer.NewMockDeployer(ctrl)
	sub := NewMockSubstrateClient(ctrl)
	gw := GatewayNameDeployer{
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		Node: 10,
		Gw: workloads.GatewayNameProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		deployer:         deployer,
		NodeDeploymentID: map[uint32]uint64{10: 100},
		NameContractID:   200,
	}
	dls, err := gw.GenerateVersionlessDeployments(context.Background())
	assert.NoError(t, err)
	deployer.EXPECT().Deploy(
		gomock.Any(),
		sub,
		map[uint32]uint64{10: 100},
		dls,
	).Return(map[uint32]uint64{10: 100}, nil)
	sub.EXPECT().
		GetContract(uint64(200)).
		Return(&substrate.Contract{
			State: substrate.ContractState{IsCreated: true},
			ContractType: substrate.ContractType{
				IsNameContract: true,
				NameContract:   substrate.NameContract{Name: "name"},
			},
		}, nil)
	err = gw.Deploy(context.Background(), sub)
	assert.NoError(t, err)
	assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{uint32(10): uint64(100)})
}

func TestNameUpdateFailed(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	deployer := deployer.NewMockDeployer(ctrl)
	sub := NewMockSubstrateClient(ctrl)
	gw := GatewayNameDeployer{
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		Node: 10,
		Gw: workloads.GatewayNameProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		deployer:         deployer,
		NodeDeploymentID: map[uint32]uint64{10: 100},
		NameContractID:   200,
	}
	dls, err := gw.GenerateVersionlessDeployments(context.Background())
	assert.NoError(t, err)
	deployer.EXPECT().Deploy(
		gomock.Any(),
		sub,
		map[uint32]uint64{10: 100},
		dls,
	).Return(map[uint32]uint64{10: 100}, errors.New("error"))
	sub.EXPECT().
		GetContract(uint64(200)).
		Return(&substrate.Contract{
			State: substrate.ContractState{IsCreated: true},
			ContractType: substrate.ContractType{
				IsNameContract: true,
				NameContract:   substrate.NameContract{Name: "name"},
			},
		}, nil)

	err = gw.Deploy(context.Background(), sub)
	assert.Error(t, err)
	assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{uint32(10): uint64(100)})
	assert.Equal(t, gw.NameContractID, uint64(200))
}

func TestNameCancel(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	deployer := deployer.NewMockDeployer(ctrl)
	sub := NewMockSubstrateClient(ctrl)
	gw := GatewayNameDeployer{
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		Node: 10,
		Gw: workloads.GatewayNameProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		deployer:         deployer,
		NodeDeploymentID: map[uint32]uint64{10: 100},
		NameContractID:   200,
	}
	deployer.EXPECT().Deploy(
		gomock.Any(),
		sub,
		map[uint32]uint64{10: 100},
		map[uint32]gridtypes.Deployment{},
	).Return(map[uint32]uint64{}, nil)
	sub.EXPECT().
		CancelContract(identity, uint64(200)).
		Return(nil)

	err = gw.Cancel(context.Background(), sub)
	assert.NoError(t, err)
	assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{})
	assert.Equal(t, gw.NameContractID, uint64(0))
}

func TestNameCancelDeploymentsFailed(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	deployer := deployer.NewMockDeployer(ctrl)
	sub := NewMockSubstrateClient(ctrl)
	gw := GatewayNameDeployer{
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		Node: 10,
		Gw: workloads.GatewayNameProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		deployer:         deployer,
		NodeDeploymentID: map[uint32]uint64{10: 100},
	}
	deployer.EXPECT().Deploy(
		gomock.Any(),
		sub,
		map[uint32]uint64{10: 100},
		map[uint32]gridtypes.Deployment{},
	).Return(map[uint32]uint64{10: 100}, errors.New("error"))
	err = gw.Cancel(context.Background(), sub)
	assert.Error(t, err)
	assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{10: 100})
}

func TestNameCancelContractsFailed(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	deployer := deployer.NewMockDeployer(ctrl)
	sub := NewMockSubstrateClient(ctrl)
	gw := GatewayNameDeployer{
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		Node: 10,
		Gw: workloads.GatewayNameProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		deployer:         deployer,
		NodeDeploymentID: map[uint32]uint64{10: 100},
		NameContractID:   200,
	}
	deployer.EXPECT().Deploy(
		gomock.Any(),
		sub,
		map[uint32]uint64{10: 100},
		map[uint32]gridtypes.Deployment{},
	).Return(map[uint32]uint64{}, nil)
	sub.EXPECT().
		CancelContract(identity, uint64(200)).
		Return(errors.New("error"))

	err = gw.Cancel(context.Background(), sub)
	assert.Error(t, err)
	assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{})
	assert.Equal(t, gw.NameContractID, uint64(200))
}

func TestNameSyncContracts(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	sub := NewMockSubstrateClient(ctrl)
	gw := GatewayNameDeployer{
		ID: "123",
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		Node: 10,
		Gw: workloads.GatewayNameProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		NodeDeploymentID: map[uint32]uint64{10: 100},
		NameContractID:   200,
	}
	sub.EXPECT().GetContract(
		uint64(100),
	).Return(&substrate.Contract{
		State: substrate.ContractState{
			IsCreated: true,
		},
	}, nil)
	sub.EXPECT().GetContract(
		uint64(200),
	).Return(&substrate.Contract{
		State: substrate.ContractState{
			IsCreated: true,
		},
	}, nil)
	err = gw.syncContracts(context.Background(), sub)
	assert.NoError(t, err)
	assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{10: 100})
	assert.Equal(t, gw.ID, "123")
}

// func TestSyncDeletedContracts(t *testing.T) {
// 	ctrl := gomock.NewController(t)

// 	defer ctrl.Finish()

// 	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
// 	assert.NoError(t, err)
// 	sub := NewMockSubstrateClient(ctrl)
// 	gw := GatewayFQDNDeployer{
// 		ID: "123",
// 		APIClient: &apiClient{
// 			identity: identity,
// 			twin_id:  11,
// 		},
// 		Node: 10,
// 		Gw: workloads.GatewayFQDNProxy{
// 			Name:           "name",
// 			TLSPassthrough: false,
// 			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
// 			FQDN:           "name.com",
// 		},
// 		NodeDeploymentID: map[uint32]uint64{10: 100},
// 	}
// 	sub.EXPECT().GetContract(
// 		uint64(100),
// 	).Return(&substrate.Contract{
// 		State: substrate.ContractState{
// 			IsCreated: false,
// 		},
// 	}, nil)
// 	err = gw.syncContracts(context.Background(), sub)
// 	assert.NoError(t, err)
// 	assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{})
// 	assert.Equal(t, gw.ID, "")
// }

// func TestSyncContractsFailure(t *testing.T) {
// 	ctrl := gomock.NewController(t)

// 	defer ctrl.Finish()

// 	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
// 	assert.NoError(t, err)
// 	sub := NewMockSubstrateClient(ctrl)
// 	gw := GatewayFQDNDeployer{
// 		ID: "123",
// 		APIClient: &apiClient{
// 			identity: identity,
// 			twin_id:  11,
// 		},
// 		Node: 10,
// 		Gw: workloads.GatewayFQDNProxy{
// 			Name:           "name",
// 			TLSPassthrough: false,
// 			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
// 			FQDN:           "name.com",
// 		},
// 		NodeDeploymentID: map[uint32]uint64{10: 100},
// 	}
// 	sub.EXPECT().GetContract(
// 		uint64(100),
// 	).Return(nil, errors.New("123"))
// 	err = gw.syncContracts(context.Background(), sub)
// 	assert.Error(t, err)
// 	assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{10: 100})
// 	assert.Equal(t, gw.ID, "123")
// }

// func TestSyncFailureInContract(t *testing.T) {
// 	ctrl := gomock.NewController(t)

// 	defer ctrl.Finish()

// 	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
// 	deployer := deployer.NewMockDeployer(ctrl)
// 	assert.NoError(t, err)
// 	sub := NewMockSubstrateClient(ctrl)
// 	gw := GatewayFQDNDeployer{
// 		ID: "123",
// 		APIClient: &apiClient{
// 			identity: identity,
// 			twin_id:  11,
// 		},
// 		Node: 10,
// 		Gw: workloads.GatewayFQDNProxy{
// 			Name:           "name",
// 			TLSPassthrough: false,
// 			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
// 			FQDN:           "name.com",
// 		},
// 		NodeDeploymentID: map[uint32]uint64{10: 100},
// 		deployer:         deployer,
// 	}
// 	sub.EXPECT().GetContract(
// 		uint64(100),
// 	).Return(nil, errors.New("123"))
// 	err = gw.sync(context.Background(), sub)
// 	assert.Error(t, err)
// 	assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{10: 100})
// 	assert.Equal(t, gw.ID, "123")
// }

// func TestSync(t *testing.T) {
// 	ctrl := gomock.NewController(t)

// 	defer ctrl.Finish()

// 	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
// 	deployer := deployer.NewMockDeployer(ctrl)
// 	pool := NewMockNodeClientCollection(ctrl)
// 	cl := NewMockClient(ctrl)
// 	assert.NoError(t, err)
// 	sub := NewMockSubstrateClient(ctrl)
// 	gw := GatewayFQDNDeployer{
// 		ID: "123",
// 		APIClient: &apiClient{
// 			identity: identity,
// 			twin_id:  11,
// 		},
// 		Node: 10,
// 		Gw: workloads.GatewayFQDNProxy{
// 			Name:           "name",
// 			TLSPassthrough: false,
// 			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
// 			FQDN:           "name.com",
// 		},
// 		NodeDeploymentID: map[uint32]uint64{10: 100},
// 		deployer:         deployer,
// 		ncPool:           pool,
// 	}
// 	dls, err := gw.GenerateVersionlessDeployments(context.Background())
// 	assert.NoError(t, err)
// 	dl := dls[10]
// 	dl.Workloads[0].Result.State = gridtypes.StateOk
// 	dl.Workloads[0].Result.Data, err = json.Marshal(zos.GatewayFQDNResult{})
// 	assert.NoError(t, err)
// 	sub.EXPECT().GetContract(
// 		uint64(100),
// 	).Return(&substrate.Contract{
// 		State: substrate.ContractState{
// 			IsCreated: true,
// 		},
// 	}, nil)
// 	pool.EXPECT().
// 		GetNodeClient(sub, uint32(10)).
// 		Return(client.NewNodeClient(12, cl), nil)
// 	cl.EXPECT().
// 		Call(gomock.Any(), uint32(12), "zos.deployment.get", gomock.Any(), gomock.Any()).
// 		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error {
// 			// TODO: check argument has correct deployment id
// 			*result.(*gridtypes.Deployment) = dl
// 			fmt.Printf("%+v", dl)
// 			return nil
// 		})
// 	gw.Gw.FQDN = "123"
// 	err = gw.sync(context.Background(), sub)
// 	assert.NoError(t, err)
// 	assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{10: 100})
// 	assert.Equal(t, gw.ID, "123")
// 	assert.Equal(t, gw.Gw.FQDN, "name.com")
// }

// func TestSyncDeletedWorkload(t *testing.T) {
// 	ctrl := gomock.NewController(t)

// 	defer ctrl.Finish()

// 	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
// 	deployer := deployer.NewMockDeployer(ctrl)
// 	pool := NewMockNodeClientCollection(ctrl)
// 	cl := NewMockClient(ctrl)
// 	assert.NoError(t, err)
// 	sub := NewMockSubstrateClient(ctrl)
// 	gw := GatewayFQDNDeployer{
// 		ID: "123",
// 		APIClient: &apiClient{
// 			identity: identity,
// 			twin_id:  11,
// 		},
// 		Node: 10,
// 		Gw: workloads.GatewayFQDNProxy{
// 			Name:           "name",
// 			TLSPassthrough: false,
// 			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
// 			FQDN:           "name.com",
// 		},
// 		NodeDeploymentID: map[uint32]uint64{10: 100},
// 		deployer:         deployer,
// 		ncPool:           pool,
// 	}
// 	dls, err := gw.GenerateVersionlessDeployments(context.Background())
// 	assert.NoError(t, err)
// 	dl := dls[10]
// 	// state is deleted
// 	sub.EXPECT().GetContract(
// 		uint64(100),
// 	).Return(&substrate.Contract{
// 		State: substrate.ContractState{
// 			IsCreated: true,
// 		},
// 	}, nil)
// 	pool.EXPECT().
// 		GetNodeClient(sub, uint32(10)).
// 		Return(client.NewNodeClient(12, cl), nil)
// 	cl.EXPECT().
// 		Call(gomock.Any(), uint32(12), "zos.deployment.get", gomock.Any(), gomock.Any()).
// 		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error {
// 			// TODO: check argument has correct deployment id
// 			*result.(*gridtypes.Deployment) = dl
// 			fmt.Printf("%+v", dl)
// 			return nil
// 		})
// 	gw.Gw.FQDN = "123"
// 	err = gw.sync(context.Background(), sub)
// 	assert.NoError(t, err)
// 	assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{10: 100})
// 	assert.Equal(t, gw.ID, "123")
// 	assert.Equal(t, gw.Gw, workloads.GatewayFQDNProxy{})
// }
