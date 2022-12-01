package provider

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"sort"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	mock "github.com/threefoldtech/terraform-provider-grid/pkg/provider/mocks"
	"github.com/threefoldtech/terraform-provider-grid/pkg/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func constructTestDeployer(ctrl *gomock.Controller) DeploymentDeployer {
	pool := mock.NewMockNodeClientCollection(ctrl)
	deployer := mock.NewMockDeployer(ctrl)
	sub := mock.NewMockSubstrateExt(ctrl)
	manager := mock.NewMockManager(ctrl)
	state := mock.NewMockStateI(ctrl)
	manager.EXPECT().SubstrateExt().Return(sub, nil).AnyTimes()
	identity := mock.NewMockIdentity(ctrl)
	identity.EXPECT().PublicKey().Return([]byte("")).AnyTimes()
	return DeploymentDeployer{
		ncPool:   pool,
		deployer: deployer,
		Id:       "100",
		Node:     10,
		Disks: []workloads.Disk{
			{
				Name:        "disk1",
				Size:        1024,
				Description: "disk1_description",
			},
			{
				Name:        "disk2",
				Size:        2048,
				Description: "disk2_description",
			},
		},
		ZDBs: []workloads.ZDB{
			{
				Name:        "zdb1",
				Password:    "pass1",
				Public:      true,
				Size:        1024,
				Description: "zdb_description",
				Mode:        "data",
				IPs: []string{
					"::1",
					"::2",
				},
				Port:      9000,
				Namespace: "ns1",
			},
			{
				Name:        "zdb2",
				Password:    "pass2",
				Public:      true,
				Size:        1024,
				Description: "zdb2_description",
				Mode:        "meta",
				IPs: []string{
					"::3",
					"::4",
				},
				Port:      9001,
				Namespace: "ns2",
			},
		},
		VMs: []workloads.VM{
			{
				Name:          "vm1",
				Flist:         "https://hub.grid.tf/tf-official-apps/discourse-v4.0.flist",
				FlistChecksum: "",
				PublicIP:      true,
				PublicIP6:     true,
				Planetary:     true,
				Corex:         true,
				ComputedIP:    "5.5.5.5/24",
				ComputedIP6:   "::7/64",
				YggIP:         "::8/64",
				IP:            "10.10.10.10",
				Description:   "vm1_description",
				Cpu:           1,
				Memory:        1024,
				RootfsSize:    1024,
				Entrypoint:    "/sbin/zinit init",
				Mounts: []workloads.Mount{
					{
						DiskName:   "disk1",
						MountPoint: "/data1",
					},
					{
						DiskName:   "disk2",
						MountPoint: "/data2",
					},
				},
				Zlogs: []workloads.Zlog{
					{
						Output: "redis://codescalers1.com",
					},
					{
						Output: "redis://threefold1.io",
					},
				},
				EnvVars: map[string]string{
					"ssh_key":  "asd",
					"ssh_key2": "asd2",
				},
				NetworkName: "network",
			},
			{
				Name:          "vm2",
				Flist:         "https://hub.grid.tf/omar0.3bot/omarelawady-ubuntu-20.04.flist",
				FlistChecksum: "f0ae02b6244db3a5f842decd082c4e08",
				PublicIP:      false,
				PublicIP6:     true,
				Planetary:     true,
				Corex:         true,
				ComputedIP:    "",
				ComputedIP6:   "::7/64",
				YggIP:         "::8/64",
				IP:            "10.10.10.10",
				Description:   "vm2_description",
				Cpu:           1,
				Memory:        1024,
				RootfsSize:    1024,
				Entrypoint:    "/sbin/zinit init",
				Mounts: []workloads.Mount{
					{
						DiskName:   "disk1",
						MountPoint: "/data1",
					},
					{
						DiskName:   "disk2",
						MountPoint: "/data2",
					},
				},
				Zlogs: []workloads.Zlog{
					{
						Output: "redis://codescalers.com",
					},
					{
						Output: "redis://threefold.io",
					},
				},
				EnvVars: map[string]string{
					"ssh_key":  "asd",
					"ssh_key2": "asd2",
				},
				NetworkName: "network",
			},
		},
		QSFSs: []workloads.QSFS{
			{
				Name:                 "name1",
				Description:          "description1",
				Cache:                1024,
				MinimalShards:        4,
				ExpectedShards:       4,
				RedundantGroups:      0,
				RedundantNodes:       0,
				MaxZDBDataDirSize:    512,
				EncryptionAlgorithm:  "AES",
				EncryptionKey:        "4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af",
				CompressionAlgorithm: "snappy",
				Metadata: workloads.Metadata{
					Type:                "zdb",
					Prefix:              "hamada",
					EncryptionAlgorithm: "AES",
					EncryptionKey:       "4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af",
					Backends: workloads.Backends{
						{
							Address:   "[::10]:8080",
							Namespace: "ns1",
							Password:  "123",
						},
						{
							Address:   "[::11]:8080",
							Namespace: "ns2",
							Password:  "1234",
						},
						{
							Address:   "[::12]:8080",
							Namespace: "ns3",
							Password:  "1235",
						},
						{
							Address:   "[::13]:8080",
							Namespace: "ns4",
							Password:  "1236",
						},
					},
				},
				Groups: workloads.Groups{
					{
						Backends: workloads.Backends{
							{
								Address:   "[::110]:8080",
								Namespace: "ns5",
								Password:  "123",
							},
							{
								Address:   "[::111]:8080",
								Namespace: "ns6",
								Password:  "1234",
							},
							{
								Address:   "[::112]:8080",
								Namespace: "ns7",
								Password:  "1235",
							},
							{
								Address:   "[::113]:8080",
								Namespace: "ns8",
								Password:  "1236",
							},
						},
					},
				},
				MetricsEndpoint: "http://[::12]:9090/metrics",
			},
		},
		IPRange:     "10.10.0.0/16",
		NetworkName: "network",
		APIClient: &apiClient{
			twin_id: 20,
			manager: manager,
			state:   state,
		},
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	r, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return r
}
func musUnmarshal(bs json.RawMessage, v interface{}) {
	err := json.Unmarshal(bs, v)
	if err != nil {
		panic(err)
	}
}

func TestValidate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := constructTestDeployer(ctrl)
	network := d.NetworkName
	checksum := d.VMs[0].FlistChecksum
	d.NetworkName = network
	d.VMs[0].FlistChecksum += " "
	assert.Error(t, d.validate())
	d.VMs[0].FlistChecksum = checksum
	assert.NoError(t, d.validate())
}

func TestDeploymentSyncDeletedContract(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := constructTestDeployer(ctrl)
	id := d.Id
	subI, _ := d.APIClient.manager.SubstrateExt()
	sub := subI.(*mock.MockSubstrateExt)
	sub.EXPECT().IsValidContract(uint64(d.ID())).Return(false, nil).AnyTimes()
	assert.NoError(t, d.syncContract(sub))
	assert.Empty(t, d.Id)
	d.Id = id
	assert.NoError(t, d.sync(context.Background(), sub, d.APIClient))
	assert.Empty(t, d.Id)
	assert.Empty(t, d.VMs)
	assert.Empty(t, d.Disks)
	assert.Empty(t, d.QSFSs)
	assert.Empty(t, d.ZDBs)
}
func TestDeploymentGenerateDeployment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := constructTestDeployer(ctrl)
	state := d.APIClient.state.(*mock.MockStateI)
	netState := mock.NewMockNetworkState(ctrl)
	state.EXPECT().GetNetworkState().Return(netState)
	network := mock.NewMockNetwork(ctrl)
	netState.EXPECT().GetNetwork(d.NetworkName).Return(network)
	network.EXPECT().GetNodeIPsList(d.Node).Return([]byte{})
	dl, err := d.GenerateVersionlessDeployments(context.Background())
	assert.NoError(t, err)
	var wls []gridtypes.Workload
	wls = append(wls, d.VMs[0].GenerateVMWorkload()...)
	wls = append(wls, d.VMs[1].GenerateVMWorkload()...)
	wl, err := d.QSFSs[0].ZosWorkload()
	assert.NoError(t, err)
	wls = append(wls, wl)
	wls = append(wls, d.Disks[0].GenerateDiskWorkload())
	wls = append(wls, d.Disks[1].GenerateDiskWorkload())
	wls = append(wls, d.ZDBs[0].GenerateZDBWorkload())
	wls = append(wls, d.ZDBs[1].GenerateZDBWorkload())
	names := make(map[string]int)
	for idx, wl := range dl[d.Node].Workloads {
		names[wl.Name.String()] = idx
	}
	sort.Slice(wls, func(i, j int) bool {
		return names[wls[i].Name.String()] < names[wls[j].Name.String()]
	})
	assert.Equal(t, len(wls), len(dl[d.Node].Workloads))
	assert.Equal(t, wls, dl[d.Node].Workloads)
}

func TestDeploymentSync(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := constructTestDeployer(ctrl)
	subI, err := d.APIClient.manager.SubstrateExt()
	assert.NoError(t, err)
	sub := subI.(*mock.MockSubstrateExt)
	state := d.APIClient.state.(*mock.MockStateI)
	netState := mock.NewMockNetworkState(ctrl)
	state.EXPECT().GetNetworkState().AnyTimes().Return(netState)
	network := mock.NewMockNetwork(ctrl)
	netState.EXPECT().GetNetwork(d.NetworkName).AnyTimes().Return(network)
	network.EXPECT().GetNodeIPsList(d.Node).Return([]byte{})
	dls, err := d.GenerateVersionlessDeployments(context.Background())
	assert.NoError(t, err)
	dl := dls[d.Node]
	json.NewEncoder(log.Writer()).Encode(dl.Workloads)
	for _, zlog := range dl.ByType(zos.ZLogsType) {
		*zlog.Workload = zlog.WithResults(gridtypes.Result{
			State: gridtypes.StateOk,
		})
	}
	for _, disk := range dl.ByType(zos.ZMountType) {
		*disk.Workload = disk.WithResults(gridtypes.Result{
			State: gridtypes.StateOk,
		})
	}
	wl, err := dl.Get(gridtypes.Name(d.VMs[0].Name))
	assert.NoError(t, err)
	*wl.Workload = wl.WithResults(gridtypes.Result{
		State: gridtypes.StateOk,
		Data: mustMarshal(zos.ZMachineResult{
			IP:    d.VMs[0].IP,
			YggIP: d.VMs[0].YggIP,
		}),
	})
	dataI, err := wl.WorkloadData()
	assert.NoError(t, err)
	data := dataI.(*zos.ZMachine)
	pubip, err := dl.Get(data.Network.PublicIP)
	assert.NoError(t, err)
	*pubip.Workload = pubip.WithResults(gridtypes.Result{
		State: gridtypes.StateOk,
		Data: mustMarshal(zos.PublicIPResult{
			IP:   gridtypes.MustParseIPNet(d.VMs[0].ComputedIP),
			IPv6: gridtypes.MustParseIPNet(d.VMs[0].ComputedIP6),
		}),
	})
	wl, err = dl.Get(gridtypes.Name(d.VMs[1].Name))
	assert.NoError(t, err)
	dataI, err = wl.WorkloadData()
	assert.NoError(t, err)
	data = dataI.(*zos.ZMachine)
	pubip, err = dl.Get(data.Network.PublicIP)
	assert.NoError(t, err)
	*pubip.Workload = pubip.WithResults(gridtypes.Result{
		State: gridtypes.StateOk,
		Data: mustMarshal(zos.PublicIPResult{
			IPv6: gridtypes.MustParseIPNet(d.VMs[1].ComputedIP6),
		}),
	})
	*wl.Workload = wl.WithResults(gridtypes.Result{
		State: gridtypes.StateOk,
		Data: mustMarshal(zos.ZMachineResult{
			IP:    d.VMs[1].IP,
			YggIP: d.VMs[1].YggIP,
		}),
	})
	wl, err = dl.Get(gridtypes.Name(d.QSFSs[0].Name))
	assert.NoError(t, err)
	*wl.Workload = wl.WithResults(gridtypes.Result{
		State: gridtypes.StateOk,
		Data: mustMarshal(zos.QuatumSafeFSResult{
			MetricsEndpoint: d.QSFSs[0].MetricsEndpoint,
		}),
	})
	wl, err = dl.Get(gridtypes.Name(d.ZDBs[0].Name))
	assert.NoError(t, err)
	*wl.Workload = wl.WithResults(gridtypes.Result{
		State: gridtypes.StateOk,
		Data: mustMarshal(zos.ZDBResult{
			Namespace: d.ZDBs[0].Namespace,
			IPs:       d.ZDBs[0].IPs,
			Port:      uint(d.ZDBs[0].Port),
		}),
	})
	wl, err = dl.Get(gridtypes.Name(d.ZDBs[1].Name))
	assert.NoError(t, err)
	*wl.Workload = wl.WithResults(gridtypes.Result{
		State: gridtypes.StateOk,
		Data: mustMarshal(zos.ZDBResult{
			Namespace: d.ZDBs[1].Namespace,
			IPs:       d.ZDBs[1].IPs,
			Port:      uint(d.ZDBs[1].Port),
		}),
	})
	for i := 0; 2*i < len(dl.Workloads); i++ {
		dl.Workloads[i], dl.Workloads[len(dl.Workloads)-1-i] =
			dl.Workloads[len(dl.Workloads)-1-i], dl.Workloads[i]
	}
	sub.EXPECT().IsValidContract(uint64(100)).Return(true, nil)
	d.deployer.(*mock.MockDeployer).EXPECT().
		GetDeploymentObjects(gomock.Any(), sub, map[uint32]uint64{10: 100}).
		Return(map[uint32]gridtypes.Deployment{
			10: dl,
		}, nil)
	var cp DeploymentDeployer
	musUnmarshal(mustMarshal(d), &cp)
	network.EXPECT().DeleteDeployment(d.Node, d.Id)
	usedIPs := getUsedIPs(dl)
	network.EXPECT().SetDeploymentIPs(d.Node, d.Id, usedIPs)
	assert.NoError(t, d.sync(context.Background(), sub, d.APIClient))
	assert.Equal(t, d.VMs, cp.VMs)
	assert.Equal(t, d.Disks, cp.Disks)
	assert.Equal(t, d.QSFSs, cp.QSFSs)
	assert.Equal(t, d.ZDBs, cp.ZDBs)
	assert.Equal(t, d.Id, cp.Id)
	assert.Equal(t, d.Node, cp.Node)
}

func getUsedIPs(dl gridtypes.Deployment) []byte {
	usedIPs := []byte{}
	for _, w := range dl.Workloads {
		if !w.Result.State.IsOkay() {
			continue
		}
		if w.Type == zos.ZMachineType {
			vm, err := workloads.NewVMFromWorkloads(&w, &dl)
			if err != nil {
				log.Printf("error parsing vm: %s", err.Error())
				continue
			}

			ip := net.ParseIP(vm.IP).To4()
			usedIPs = append(usedIPs, ip[3])
		}
	}
	return usedIPs
}
