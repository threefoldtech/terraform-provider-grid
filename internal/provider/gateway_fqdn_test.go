package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/substrate-client"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	mock "github.com/threefoldtech/terraform-provider-grid/internal/provider/mocks"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
	"github.com/threefoldtech/terraform-provider-grid/pkg/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

const Words = "actress baby exhaust blind forget vintage express torch luxury symbol weird eight"

func TestValidatNodeReachable(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	sub := mock.NewMockSubstrate(ctrl)
	cl := mock.NewRMBMockClient(ctrl)
	pool := mock.NewMockNodeClientCollection(ctrl)
	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
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

	gw := GatewayFQDNDeployer{
		APIClient: &apiClient{
			identity: identity,
		},
		ncPool: pool,
		NodeID: 11,
	}
	err = gw.Validate(context.TODO(), sub)
	assert.NoError(t, err)
}

func TestGenerateDeployment(t *testing.T) {
	g := workloads.GatewayFQDNProxy{
		Name:           "name",
		TLSPassthrough: false,
		Backends:       []zos.Backend{"a", "b"},
		FQDN:           "name.com",
	}
	gw := GatewayFQDNDeployer{
		APIClient: &apiClient{
			twin_id: 11,
		},
		CapacityID: 10,
		Gw:         g,
	}
	dls, err := gw.GenerateVersionlessDeployments(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, dls, map[uint64]gridtypes.Deployment{
		10: {
			Version: 0,
			TwinID:  11,
			Workloads: []gridtypes.Workload{
				{
					Version: 0,
					Type:    zos.GatewayFQDNProxyType,
					Name:    gridtypes.Name(g.Name),
					Data: gridtypes.MustMarshal(zos.GatewayFQDNProxy{
						TLSPassthrough: g.TLSPassthrough,
						Backends:       g.Backends,
						FQDN:           gw.Gw.FQDN,
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

func TestDeploy(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	deployer := mock.NewMockDeployer(ctrl)
	sub := mock.NewMockSubstrate(ctrl)
	cl := mock.NewRMBMockClient(ctrl)
	pool := mock.NewMockNodeClientCollection(ctrl)
	gw := GatewayFQDNDeployer{
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		CapacityID: 10,
		NodeID:     1,
		Gw: workloads.GatewayFQDNProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		deployer: deployer,
		ncPool:   pool,
	}
	dls, err := gw.GenerateVersionlessDeployments(context.Background())
	assert.NoError(t, err)
	pool.EXPECT().
		GetNodeClient(sub, uint32(1)).
		Return(client.NewNodeClient(12, cl), nil)
	cl.EXPECT().Call(
		gomock.Any(),
		uint32(12),
		"zos.network.interfaces",
		gomock.Any(),
		gomock.Any(),
	).Return(nil)
	deployer.EXPECT().Deploy(
		gomock.Any(),
		sub,
		nil,
		dls,
	).Return(map[uint64]uint64{10: 100}, nil)
	err = gw.Deploy(context.Background(), sub)
	assert.NoError(t, err)
	assert.NotEmpty(t, gw.ID)
	assert.Equal(t, gw.CapacityDeploymentMap, map[uint64]uint64{uint64(10): uint64(100)})
}

func TestUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	deployer := mock.NewMockDeployer(ctrl)
	sub := mock.NewMockSubstrate(ctrl)
	cl := mock.NewRMBMockClient(ctrl)
	pool := mock.NewMockNodeClientCollection(ctrl)
	gw := GatewayFQDNDeployer{
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		CapacityID: 10,
		NodeID:     1,
		Gw: workloads.GatewayFQDNProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		deployer:              deployer,
		ncPool:                pool,
		CapacityDeploymentMap: map[uint64]uint64{10: 100},
	}
	dls, err := gw.GenerateVersionlessDeployments(context.Background())
	assert.NoError(t, err)
	deployer.EXPECT().Deploy(
		gomock.Any(),
		sub,
		map[uint64]uint64{10: 100},
		dls,
	).Return(map[uint64]uint64{uint64(10): uint64(100)}, nil)
	pool.EXPECT().
		GetNodeClient(sub, uint32(1)).
		Return(client.NewNodeClient(12, cl), nil)
	cl.EXPECT().Call(
		gomock.Any(),
		uint32(12),
		"zos.network.interfaces",
		gomock.Any(),
		gomock.Any(),
	).Return(nil)
	err = gw.Deploy(context.Background(), sub)
	assert.NoError(t, err)
	assert.Equal(t, gw.CapacityDeploymentMap, map[uint64]uint64{uint64(10): uint64(100)})
}
func TestUpdateFailed(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	deployer := mock.NewMockDeployer(ctrl)
	sub := mock.NewMockSubstrate(ctrl)
	cl := mock.NewRMBMockClient(ctrl)
	pool := mock.NewMockNodeClientCollection(ctrl)

	gw := GatewayFQDNDeployer{
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		CapacityID: 10,
		NodeID:     1,
		Gw: workloads.GatewayFQDNProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		deployer:              deployer,
		ncPool:                pool,
		CapacityDeploymentMap: map[uint64]uint64{10: 100},
	}
	dls, err := gw.GenerateVersionlessDeployments(context.Background())
	assert.NoError(t, err)
	deployer.EXPECT().Deploy(
		gomock.Any(),
		sub,
		map[uint64]uint64{10: 100},
		dls,
	).Return(map[uint64]uint64{10: 100}, errors.New("error"))
	pool.EXPECT().
		GetNodeClient(sub, uint32(1)).
		Return(client.NewNodeClient(12, cl), nil)
	cl.EXPECT().Call(
		gomock.Any(),
		uint32(12),
		"zos.network.interfaces",
		gomock.Any(),
		gomock.Any(),
	).Return(nil)

	err = gw.Deploy(context.Background(), sub)
	assert.Error(t, err)
	assert.Equal(t, gw.CapacityDeploymentMap, map[uint64]uint64{uint64(10): uint64(100)})
}

func TestCancel(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	deployer := mock.NewMockDeployer(ctrl)
	sub := mock.NewMockSubstrate(ctrl)
	gw := GatewayFQDNDeployer{
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		CapacityID: 10,
		Gw: workloads.GatewayFQDNProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		deployer:              deployer,
		CapacityDeploymentMap: map[uint64]uint64{10: 100},
	}
	deployer.EXPECT().Deploy(
		gomock.Any(),
		sub,
		map[uint64]uint64{10: 100},
		map[uint64]gridtypes.Deployment{},
	).Return(map[uint64]uint64{}, nil)
	err = gw.Cancel(context.Background(), sub)
	assert.NoError(t, err)
	assert.Equal(t, gw.CapacityDeploymentMap, map[uint64]uint64{})
}

func TestCancelFailed(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	deployer := mock.NewMockDeployer(ctrl)
	sub := mock.NewMockSubstrate(ctrl)
	gw := GatewayFQDNDeployer{
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		CapacityID: 10,
		Gw: workloads.GatewayFQDNProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		deployer:              deployer,
		CapacityDeploymentMap: map[uint64]uint64{10: 100},
	}
	deployer.EXPECT().Deploy(
		gomock.Any(),
		sub,
		map[uint64]uint64{10: 100},
		map[uint64]gridtypes.Deployment{},
	).Return(map[uint64]uint64{10: 100}, errors.New("error"))
	err = gw.Cancel(context.Background(), sub)
	assert.Error(t, err)
	assert.Equal(t, gw.CapacityDeploymentMap, map[uint64]uint64{10: 100})
}

func TestSyncContracts(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	sub := mock.NewMockSubstrate(ctrl)
	gw := GatewayFQDNDeployer{
		ID: "123",
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		CapacityID: 10,
		Gw: workloads.GatewayFQDNProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		CapacityDeploymentMap: map[uint64]uint64{10: 100},
	}
	sub.EXPECT().GetContract(
		gw.CapacityID,
	).Return(&substrate.Contract{State: substrate.ContractState{IsCreated: true}}, nil)
	sub.EXPECT().GetDeployment(
		gw.CapacityDeploymentMap[gw.CapacityID],
	).Return(&substrate.Deployment{}, nil)
	err = gw.syncContracts(context.Background(), sub)
	assert.NoError(t, err)
	assert.Equal(t, gw.CapacityDeploymentMap, map[uint64]uint64{10: 100})
	assert.Equal(t, gw.ID, "123")
}

// func TestSyncDeletedContracts(t *testing.T) {
// 	ctrl := gomock.NewController(t)

// 	defer ctrl.Finish()

// 	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
// 	assert.NoError(t, err)
// 	sub := mock.NewMockSubstrate(ctrl)
// 	gw := GatewayFQDNDeployer{
// 		ID: "123",
// 		APIClient: &apiClient{
// 			identity: identity,
// 			twin_id:  11,
// 		},
// 		CapacityID: 10,
// 		Gw: workloads.GatewayFQDNProxy{
// 			Name:           "name",
// 			TLSPassthrough: false,
// 			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
// 			FQDN:           "name.com",
// 		},
// 		CapacityDeploymentMap: map[uint64]uint64{10: 100},
// 	}

// 	sub.EXPECT().DeleteInvalidContracts(
// 		gw.CapacityDeploymentMap,
// 	).DoAndReturn(func(contracts map[uint64]uint64) error {
// 		delete(contracts, 10)
// 		return nil
// 	})
// 	err = gw.syncContracts(context.Background(), sub)
// 	assert.NoError(t, err)
// 	assert.Equal(t, gw.CapacityDeploymentMap, map[uint64]uint64{})
// 	assert.Equal(t, gw.ID, "")
// }

func TestSyncContractsFailure(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	sub := mock.NewMockSubstrate(ctrl)
	gw := GatewayFQDNDeployer{
		ID: "123",
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		CapacityID: 10,
		Gw: workloads.GatewayFQDNProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		CapacityDeploymentMap: map[uint64]uint64{10: 100},
	}

	sub.EXPECT().GetContract(
		gw.CapacityID,
	).Return(nil, errors.New("123"))
	err = gw.syncContracts(context.Background(), sub)
	assert.Error(t, err)
	assert.Equal(t, gw.CapacityDeploymentMap, map[uint64]uint64{10: 100})
	assert.Equal(t, gw.ID, "123")
}

func TestSyncFailureInContract(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	deployer := mock.NewMockDeployer(ctrl)
	assert.NoError(t, err)
	sub := mock.NewMockSubstrate(ctrl)
	gw := GatewayFQDNDeployer{
		ID: "123",
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		CapacityID: 10,
		Gw: workloads.GatewayFQDNProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		CapacityDeploymentMap: map[uint64]uint64{10: 100},
		deployer:              deployer,
	}

	sub.EXPECT().GetContract(
		gw.CapacityID,
	).Return(nil, errors.New("123"))
	err = gw.sync(context.Background(), sub, gw.APIClient)
	assert.Error(t, err)
	assert.Equal(t, gw.CapacityDeploymentMap, map[uint64]uint64{10: 100})
	assert.Equal(t, gw.ID, "123")
}

func TestSync(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	deployer := mock.NewMockDeployer(ctrl)
	pool := mock.NewMockNodeClientCollection(ctrl)
	sub := mock.NewMockSubstrate(ctrl)
	gw := GatewayFQDNDeployer{
		ID: "123",
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		CapacityID: 10,
		Gw: workloads.GatewayFQDNProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		CapacityDeploymentMap: map[uint64]uint64{10: 100},
		deployer:              deployer,
		ncPool:                pool,
	}
	dls, err := gw.GenerateVersionlessDeployments(context.Background())
	assert.NoError(t, err)
	dl := dls[10]
	dl.Workloads[0].Result.State = gridtypes.StateOk
	dl.Workloads[0].Result.Data, err = json.Marshal(zos.GatewayFQDNResult{})
	assert.NoError(t, err)

	sub.EXPECT().GetContract(
		gw.CapacityID,
	).Return(&substrate.Contract{State: substrate.ContractState{IsCreated: true}}, nil)
	sub.EXPECT().GetDeployment(
		gw.CapacityDeploymentMap[gw.CapacityID],
	).Return(&substrate.Deployment{}, nil)

	deployer.EXPECT().
		GetDeploymentObjects(gomock.Any(), sub, map[uint64]uint64{10: 100}).
		DoAndReturn(func(ctx context.Context, _ subi.Substrate, _ map[uint64]uint64) (map[uint64]gridtypes.Deployment, error) {
			return map[uint64]gridtypes.Deployment{10: dl}, nil
		})
	gw.Gw.FQDN = "123"
	err = gw.sync(context.Background(), sub, gw.APIClient)
	assert.NoError(t, err)
	assert.Equal(t, gw.CapacityDeploymentMap, map[uint64]uint64{10: 100})
	assert.Equal(t, gw.ID, "123")
	assert.Equal(t, gw.Gw.FQDN, "name.com")
}

func TestSyncDeletedWorkload(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	identity, err := substrate.NewIdentityFromEd25519Phrase(Words)
	assert.NoError(t, err)
	deployer := mock.NewMockDeployer(ctrl)
	pool := mock.NewMockNodeClientCollection(ctrl)
	sub := mock.NewMockSubstrate(ctrl)
	gw := GatewayFQDNDeployer{
		ID: "123",
		APIClient: &apiClient{
			identity: identity,
			twin_id:  11,
		},
		CapacityID: 10,
		Gw: workloads.GatewayFQDNProxy{
			Name:           "name",
			TLSPassthrough: false,
			Backends:       []zos.Backend{"https://1.1.1.1", "http://2.2.2.2"},
			FQDN:           "name.com",
		},
		CapacityDeploymentMap: map[uint64]uint64{10: 100},
		deployer:              deployer,
		ncPool:                pool,
	}
	dls, err := gw.GenerateVersionlessDeployments(context.Background())
	assert.NoError(t, err)
	dl := dls[10]
	// state is deleted

	sub.EXPECT().GetContract(
		gw.CapacityID,
	).Return(&substrate.Contract{State: substrate.ContractState{IsCreated: true}}, nil)
	sub.EXPECT().GetDeployment(
		gw.CapacityDeploymentMap[gw.CapacityID],
	).Return(&substrate.Deployment{}, nil)
	deployer.EXPECT().
		GetDeploymentObjects(gomock.Any(), sub, map[uint64]uint64{10: 100}).
		DoAndReturn(func(ctx context.Context, _ subi.Substrate, _ map[uint64]uint64) (map[uint64]gridtypes.Deployment, error) {
			return map[uint64]gridtypes.Deployment{10: dl}, nil
		})
	gw.Gw.FQDN = "123"
	err = gw.sync(context.Background(), sub, gw.APIClient)
	assert.NoError(t, err)
	assert.Equal(t, gw.CapacityDeploymentMap, map[uint64]uint64{10: 100})
	assert.Equal(t, gw.ID, "123")
	assert.Equal(t, gw.Gw, workloads.GatewayFQDNProxy{})
}
