package deployer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
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
const twinID = 11

var identity, _ = substrate.NewIdentityFromEd25519Phrase(Words)

func deployment1(identity substrate.Identity, TLSPassthrough bool, version uint32) gridtypes.Deployment {
	dl := workloads.NewDeployment(uint32(twinID))
	dl.Version = version
	gw := workloads.GatewayNameProxy{
		Name:           "name",
		TLSPassthrough: TLSPassthrough,
		Backends:       []zos.Backend{"http://1.1.1.1"},
	}
	dl.Workloads = append(dl.Workloads, gw.ZosWorkload())
	dl.Workloads[0].Version = version
	err := dl.Sign(twinID, identity)
	if err != nil {
		panic(err)
	}
	return dl
}

func deployment2(identity substrate.Identity) gridtypes.Deployment {
	dl := workloads.NewDeployment(uint32(twinID))
	gw := workloads.GatewayFQDNProxy{
		Name:     "fqdn",
		FQDN:     "a.b.com",
		Backends: []zos.Backend{"http://1.1.1.1"},
	}
	dl.Workloads = append(dl.Workloads, gw.ZosWorkload())
	err := dl.Sign(twinID, identity)
	if err != nil {
		panic(err)
	}

	return dl
}
func hash(dl *gridtypes.Deployment) string {
	hash, err := dl.ChallengeHash()
	if err != nil {
		panic(err)
	}
	hashHex := hex.EncodeToString(hash)
	return hashHex
}

type EmptyValidator struct{}

func (d *EmptyValidator) Validate(ctx context.Context, sub subi.SubstrateExt, oldDeployments map[uint32]gridtypes.Deployment, newDeployments map[uint32]gridtypes.Deployment) error {
	return nil
}
func TestCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	gridClient := mock.NewMockClient(ctrl)
	cl := mock.NewRMBMockClient(ctrl)
	sub := mock.NewMockSubstrateExt(ctrl)
	ncPool := mock.NewMockNodeClientCollection(ctrl)
	deployer := NewDeployer(
		identity,
		11,
		gridClient,
		ncPool,
		true,
		nil,
		"",
	)
	dl1, dl2 := deployment1(identity, true, 0), deployment2(identity)
	newDls := map[uint32]gridtypes.Deployment{
		10: dl1,
		20: dl2,
	}
	dl1.ContractID = 100
	dl2.ContractID = 200
	var solutionProvider *uint64
	*solutionProvider = 0
	sub.EXPECT().
		CreateNodeContract(
			identity,
			uint32(10),
			nil,
			hash(&dl1),
			uint32(0),
			solutionProvider,
		).Return(uint64(100), nil)
	sub.EXPECT().
		CreateNodeContract(
			identity,
			uint32(20),
			nil,
			hash(&dl2),
			uint32(0),
			solutionProvider,
		).Return(uint64(200), nil)
	ncPool.EXPECT().
		GetNodeClient(sub, uint32(10)).
		Return(client.NewNodeClient(13, cl), nil)
	ncPool.EXPECT().
		GetNodeClient(sub, uint32(20)).
		Return(client.NewNodeClient(23, cl), nil)
	cl.EXPECT().
		Call(gomock.Any(), uint32(13), "zos.deployment.deploy", dl1, gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			dl1.Workloads[0].Result.State = gridtypes.StateOk
			dl1.Workloads[0].Result.Data, _ = json.Marshal(zos.GatewayProxyResult{})
			return nil
		})
	cl.EXPECT().
		Call(gomock.Any(), uint32(23), "zos.deployment.deploy", dl2, gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			dl2.Workloads[0].Result.State = gridtypes.StateOk
			dl2.Workloads[0].Result.Data, _ = json.Marshal(zos.GatewayFQDNResult{})
			return nil
		})
	cl.EXPECT().
		Call(gomock.Any(), uint32(13), "zos.deployment.get", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			var res *gridtypes.Deployment = result.(*gridtypes.Deployment)
			*res = dl1
			return nil
		})
	cl.EXPECT().
		Call(gomock.Any(), uint32(23), "zos.deployment.get", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			var res *gridtypes.Deployment = result.(*gridtypes.Deployment)
			*res = dl2
			return nil
		})
	deployer.(*DeployerImpl).validator = &EmptyValidator{}
	contracts, err := deployer.Deploy(context.Background(), sub, nil, newDls)
	assert.NoError(t, err)
	assert.Equal(t, contracts, map[uint32]uint64{10: 100, 20: 200})
}

func TestUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	gridClient := mock.NewMockClient(ctrl)
	cl := mock.NewRMBMockClient(ctrl)
	sub := mock.NewMockSubstrateExt(ctrl)
	ncPool := mock.NewMockNodeClientCollection(ctrl)
	deployer := NewDeployer(
		identity,
		11,
		gridClient,
		ncPool,
		true,
		nil,
		"",
	)
	dl1, dl2 := deployment1(identity, false, 0), deployment1(identity, true, 1)
	newDls := map[uint32]gridtypes.Deployment{
		10: dl2,
	}

	dl1.ContractID = 100
	dl2.ContractID = 100
	sub.EXPECT().
		UpdateNodeContract(
			identity,
			uint64(100),
			nil,
			hash(&dl2),
		).Return(uint64(100), nil)
	ncPool.EXPECT().
		GetNodeClient(sub, uint32(10)).
		Return(client.NewNodeClient(13, cl), nil).AnyTimes()
	cl.EXPECT().
		Call(gomock.Any(), uint32(13), "zos.deployment.update", dl2, gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			dl1.Workloads[0].Result.State = gridtypes.StateOk
			dl1.Workloads[0].Result.Data, _ = json.Marshal(zos.GatewayProxyResult{})
			dl1.Version = 1
			dl1.Workloads[0].Version = 1
			return nil
		})
	cl.EXPECT().
		Call(gomock.Any(), uint32(13), "zos.deployment.get", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			var res *gridtypes.Deployment = result.(*gridtypes.Deployment)
			*res = dl1
			return nil
		}).AnyTimes()
	deployer.(*DeployerImpl).validator = &EmptyValidator{}
	contracts, err := deployer.Deploy(context.Background(), sub, map[uint32]uint64{10: 100}, newDls)
	assert.NoError(t, err)
	assert.Equal(t, contracts, map[uint32]uint64{10: 100})
	assert.Equal(t, dl1.Version, dl2.Version)
	assert.Equal(t, dl1.Workloads[0].Version, dl2.Workloads[0].Version)
}

func TestCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	gridClient := mock.NewMockClient(ctrl)
	cl := mock.NewRMBMockClient(ctrl)
	sub := mock.NewMockSubstrateExt(ctrl)
	ncPool := mock.NewMockNodeClientCollection(ctrl)
	deployer := NewDeployer(
		identity,
		11,
		gridClient,
		ncPool,
		true,
		nil,
		"",
	)
	dl1 := deployment1(identity, false, 0)
	dl1.ContractID = 100
	sub.EXPECT().
		EnsureContractCanceled(
			identity,
			uint64(100),
		).Return(nil)
	ncPool.EXPECT().
		GetNodeClient(sub, uint32(10)).
		Return(client.NewNodeClient(13, cl), nil).AnyTimes()
	cl.EXPECT().
		Call(gomock.Any(), uint32(13), "zos.deployment.get", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			var res *gridtypes.Deployment = result.(*gridtypes.Deployment)
			*res = dl1
			return nil
		})
	cl.EXPECT().
		Call(gomock.Any(), uint32(13), "zos.deployment.delete", gomock.Any(), gomock.Any()).
		Return(nil)
	deployer.(*DeployerImpl).validator = &EmptyValidator{}
	contracts, err := deployer.Deploy(context.Background(), sub, map[uint32]uint64{10: 100}, nil)
	assert.NoError(t, err)
	assert.Equal(t, contracts, map[uint32]uint64{})
}

func TestCocktail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	gridClient := mock.NewMockClient(ctrl)
	cl := mock.NewRMBMockClient(ctrl)
	sub := mock.NewMockSubstrateExt(ctrl)
	ncPool := mock.NewMockNodeClientCollection(ctrl)
	deployer := NewDeployer(
		identity,
		11,
		gridClient,
		ncPool,
		true,
		nil,
		"",
	)
	g := workloads.GatewayFQDNProxy{Name: "f", FQDN: "test.com", Backends: []zos.Backend{"http://1.1.1.1:10"}}
	dl1 := deployment1(identity, false, 0)
	dl2, dl3 := deployment1(identity, false, 0), deployment1(identity, true, 1)
	dl5, dl6 := deployment1(identity, true, 0), deployment1(identity, true, 0)
	dl2.Workloads = append(dl2.Workloads, g.ZosWorkload())
	dl3.Workloads = append(dl3.Workloads, g.ZosWorkload())
	assert.NoError(t, dl2.Sign(twinID, identity))
	assert.NoError(t, dl3.Sign(twinID, identity))
	dl4 := deployment1(identity, false, 0)
	dl1.ContractID = 100
	dl2.ContractID = 200
	dl3.ContractID = 200
	dl4.ContractID = 300
	oldDls := map[uint32]uint64{
		10: 100,
		20: 200,
		40: 400,
	}
	newDls := map[uint32]gridtypes.Deployment{
		20: dl3,
		30: dl4,
		40: dl6,
	}
	var solutionProvider *uint64
	*solutionProvider = 0
	sub.EXPECT().
		CreateNodeContract(
			identity,
			uint32(30),
			nil,
			hash(&dl4),
			uint32(0),
			solutionProvider,
		).Return(uint64(300), nil)

	sub.EXPECT().
		UpdateNodeContract(
			identity,
			uint64(200),
			nil,
			hash(&dl3),
		).Return(uint64(200), nil)

	sub.EXPECT().
		EnsureContractCanceled(
			identity,
			uint64(100),
		).Return(nil)
	ncPool.EXPECT().
		GetNodeClient(sub, uint32(10)).
		Return(client.NewNodeClient(13, cl), nil).AnyTimes()
	ncPool.EXPECT().
		GetNodeClient(sub, uint32(20)).
		Return(client.NewNodeClient(23, cl), nil).AnyTimes()
	ncPool.EXPECT().
		GetNodeClient(sub, uint32(30)).
		Return(client.NewNodeClient(33, cl), nil).AnyTimes()
	ncPool.EXPECT().
		GetNodeClient(sub, uint32(40)).
		Return(client.NewNodeClient(43, cl), nil).AnyTimes()
	cl.EXPECT().
		Call(gomock.Any(), uint32(13), "zos.deployment.get", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			var res *gridtypes.Deployment = result.(*gridtypes.Deployment)
			*res = dl1
			return nil
		}).AnyTimes()
	cl.EXPECT().
		Call(gomock.Any(), uint32(23), "zos.deployment.get", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			var res *gridtypes.Deployment = result.(*gridtypes.Deployment)
			*res = dl2
			return nil
		}).AnyTimes()
	cl.EXPECT().
		Call(gomock.Any(), uint32(33), "zos.deployment.get", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			var res *gridtypes.Deployment = result.(*gridtypes.Deployment)
			*res = dl4
			return nil
		}).AnyTimes()
	cl.EXPECT().
		Call(gomock.Any(), uint32(43), "zos.deployment.get", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			var res *gridtypes.Deployment = result.(*gridtypes.Deployment)
			*res = dl5
			return nil
		}).AnyTimes()
	cl.EXPECT().
		Call(gomock.Any(), uint32(13), "zos.deployment.delete", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			dl1.Workloads[0].Result.State = gridtypes.StateDeleted
			dl1.Workloads[0].Result.Data, _ = json.Marshal(zos.GatewayProxyResult{})
			return nil
		})
	cl.EXPECT().
		Call(gomock.Any(), uint32(23), "zos.deployment.update", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			dl2.Workloads = dl3.Workloads
			dl2.Version = 1
			dl2.Workloads[0].Version = 1
			dl2.Workloads[0].Result.State = gridtypes.StateOk
			dl2.Workloads[0].Result.Data, _ = json.Marshal(zos.GatewayProxyResult{})
			dl2.Workloads[1].Result.State = gridtypes.StateOk
			dl2.Workloads[1].Result.Data, _ = json.Marshal(zos.GatewayProxyResult{})
			return nil
		})
	cl.EXPECT().
		Call(gomock.Any(), uint32(33), "zos.deployment.deploy", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			dl4.Workloads[0].Result.State = gridtypes.StateOk
			dl4.Workloads[0].Result.Data, _ = json.Marshal(zos.GatewayProxyResult{})
			return nil
		})
	deployer.(*DeployerImpl).validator = &EmptyValidator{}
	contracts, err := deployer.Deploy(context.Background(), sub, oldDls, newDls)
	assert.NoError(t, err)
	assert.Equal(t, contracts, map[uint32]uint64{
		20: 200,
		30: 300,
		40: 400,
	})
}
