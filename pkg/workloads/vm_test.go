package workloads

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func vmInterface() map[string]interface{} {
	return map[string]interface{}{
		"name":           "n",
		"flist":          "https://hub.grid.tf/omar0.3bot/omarelawady-ubuntu-20.04.flist",
		"flist_checksum": "f0ae02b6244db3a5f842decd082c4e08",
		"publicip":       true,
		"publicip6":      true,
		"computedip":     "189.0.0.12/24",
		"computedip6":    "::/64",
		"ip":             "10.0.0.1",
		"cpu":            1,
		"description":    "des",
		"memory":         1024,
		"rootfs_size":    1024,
		"entrypoint":     "/sbin/zinit",
		"mounts": []interface{}{
			map[string]interface{}{
				"disk_name":   "data",
				"mount_point": "/data",
			},
			map[string]interface{}{
				"disk_name":   "data1",
				"mount_point": "/data1",
			},
		},
		"env_vars": map[string]interface{}{
			"ssh_key":  "asd",
			"ssh_key2": "asd2",
		},
		"planetary": true,
		"corex":     true,
		"ygg_ip":    "::/64",
		"zlogs": []interface{}{
			"redis://codescalers.com",
			"redis://threefold.io",
		},
	}
}
func vmObj() *VM {
	return &VM{
		Name:          "n",
		Flist:         "https://hub.grid.tf/omar0.3bot/omarelawady-ubuntu-20.04.flist",
		FlistChecksum: "f0ae02b6244db3a5f842decd082c4e08",
		PublicIP:      true,
		PublicIP6:     true,
		ComputedIP:    "189.0.0.12/24",
		ComputedIP6:   "::/64",
		IP:            "10.0.0.1",
		Cpu:           1,
		Description:   "des",
		Memory:        1024,
		RootfsSize:    1024,
		Entrypoint:    "/sbin/zinit",
		Mounts: []Mount{
			{
				DiskName:   "data",
				MountPoint: "/data",
			},
			{
				DiskName:   "data1",
				MountPoint: "/data1",
			},
		},
		EnvVars: map[string]string{
			"ssh_key":  "asd",
			"ssh_key2": "asd2",
		},
		Planetary: true,
		Corex:     true,
		YggIP:     "::/64",
		Zlogs: []Zlog{
			{
				Output: "redis://codescalers.com",
			},
			{
				Output: "redis://threefold.io",
			},
		},
	}
}
func TestNewVMFromSchema(t *testing.T) {
	in := vmInterface()
	vm := NewVMFromSchema(in)
	assert.Equal(t, vm, vmObj())
}

func TestGenerateVMWorkload(t *testing.T) {
	vm := vmObj().WithNetworkName("network")
	wls := vm.GenerateVMWorkload()
	d := NewDeployment(11)
	d.Workloads = wls

	wl, err := d.Get(gridtypes.Name(vm.Name))
	assert.NoError(t, err)
	dataI, err := wl.WorkloadData()
	assert.NoError(t, err)
	pubIP := dataI.(*zos.ZMachine).Network.PublicIP
	assert.NotEmpty(t, pubIP)
	ipWl, err := d.Get(pubIP)
	assert.NoError(t, err)
	dataI, err = ipWl.WorkloadData()
	assert.NoError(t, err)
	pubip := dataI.(*zos.PublicIP)
	assert.Equal(t, pubip, &zos.PublicIP{
		V4: true,
		V6: true,
	})
	assert.Equal(t, wl.Workload, &gridtypes.Workload{
		Version:     0,
		Name:        gridtypes.Name(vm.Name),
		Type:        zos.ZMachineType,
		Description: vm.Description,
		Data: gridtypes.MustMarshal(zos.ZMachine{
			FList: vm.Flist,
			Network: zos.MachineNetwork{
				PublicIP:  pubIP,
				Planetary: vm.Planetary,
				Interfaces: []zos.MachineInterface{
					{
						Network: gridtypes.Name(vm.NetworkName),
						IP:      net.ParseIP(vm.IP),
					},
				},
			},
			Size: gridtypes.Unit(vm.RootfsSize * 1024 * 1024),
			ComputeCapacity: zos.MachineCapacity{
				CPU:    uint8(vm.Cpu),
				Memory: gridtypes.Unit(vm.Memory * 1024 * 1024),
			},
			Mounts: []zos.MachineMount{
				{
					Name:       gridtypes.Name(vm.Mounts[0].DiskName),
					Mountpoint: vm.Mounts[0].MountPoint,
				},
				{
					Name:       gridtypes.Name(vm.Mounts[1].DiskName),
					Mountpoint: vm.Mounts[1].MountPoint,
				},
			},
			Entrypoint: vm.Entrypoint,
			Env:        vm.EnvVars,
			Corex:      vm.Corex,
		}),
	})
	for _, zlog := range d.ByType(zos.ZLogsType) {
		dataI, err := zlog.Workload.WorkloadData()
		assert.NoError(t, err)
		assert.Equal(t, dataI.(*zos.ZLogs).ZMachine, gridtypes.Name(vm.Name))
	}
	assert.NoError(t, d.Valid())
}

func TestDictify(t *testing.T) {
	in := vmInterface()
	vm := vmObj()
	assert.Equal(t, in, vm.Dictify())
}

func TestValidate(t *testing.T) {
	vm := vmObj()
	assert.NoError(t, vm.Validate())
}

func TestValidateFailure(t *testing.T) {
	t.Run("Invalid_flist", func (t *testing.T) {
        vm := vmObj()
        vm.FlistChecksum += "a"
        assert.Error(t, vm.Validate())
    })

    t.Run("Invalid_CPU", func(t *testing.T) {
        vm := vmObj()
        vm.Cpu = 1000
        assert.Error(t, vm.Validate())
    })
}

func TestNewVMFromWorkloads(t *testing.T) {
	vm := vmObj().WithNetworkName("network")
	wls := vm.GenerateVMWorkload()
	d := NewDeployment(11)
	d.Workloads = wls
	wl, err := d.Get(gridtypes.Name(vm.Name))
	assert.NoError(t, err)
	dataI, err := wl.WorkloadData()
	assert.NoError(t, err)
	pubIP := dataI.(*zos.ZMachine).Network.PublicIP
	assert.NotEmpty(t, pubIP)
	ipWl, err := d.Get(pubIP)
	assert.NoError(t, err)
	pubIPRes, err := json.Marshal(zos.PublicIPResult{
		IP:      gridtypes.MustParseIPNet("189.0.0.12/24"),
		IPv6:    gridtypes.MustParseIPNet("::/64"),
		Gateway: nil,
	})
	assert.NoError(t, err)
	*ipWl.Workload = ipWl.WithResults(gridtypes.Result{
		State: gridtypes.StateOk,
		Data:  json.RawMessage(pubIPRes),
	})
	vmRes, err := json.Marshal(zos.ZMachineResult{
		IP:    "",
		ID:    "",
		YggIP: "::/64",
	})
	assert.NoError(t, err)
	*wl.Workload = wl.Workload.WithResults(gridtypes.Result{
		State: gridtypes.StateOk,
		Data:  vmRes,
	})
	for _, zlog := range d.ByType(zos.ZLogsType) {
		*zlog.Workload = zlog.WithResults(gridtypes.Result{
			State: gridtypes.StateOk,
		})
	}

	wlVM, err := NewVMFromWorkloads(wl.Workload, &d)
	vm.FlistChecksum = ""
	assert.NoError(t, err)
	assert.Equal(t, *vm, wlVM)
}

func TestMatch(t *testing.T) {
	vm := vmObj().WithNetworkName("network")
	wls := vm.GenerateVMWorkload()
	for i, j := 0, len(wls)-1; i < j; i, j = i+1, j-1 {
		wls[i], wls[j] = wls[j], wls[i]
	}
	d := NewDeployment(11)
	d.Workloads = wls
	wl, err := d.Get(gridtypes.Name(vm.Name))
	assert.NoError(t, err)
	dataI, err := wl.WorkloadData()
	assert.NoError(t, err)
	pubIP := dataI.(*zos.ZMachine).Network.PublicIP
	assert.NotEmpty(t, pubIP)
	ipWl, err := d.Get(pubIP)
	assert.NoError(t, err)
	pubIPRes, err := json.Marshal(zos.PublicIPResult{
		IP:      gridtypes.MustParseIPNet("189.0.0.12/24"),
		IPv6:    gridtypes.MustParseIPNet("::/64"),
		Gateway: nil,
	})
	assert.NoError(t, err)
	*ipWl.Workload = ipWl.WithResults(gridtypes.Result{
		State: gridtypes.StateOk,
		Data:  json.RawMessage(pubIPRes),
	})
	vmRes, err := json.Marshal(zos.ZMachineResult{
		IP:    "",
		ID:    "",
		YggIP: "::/64",
	})
	assert.NoError(t, err)
	*wl.Workload = wl.Workload.WithResults(gridtypes.Result{
		State: gridtypes.StateOk,
		Data:  vmRes,
	})
	for _, zlog := range d.ByType(zos.ZLogsType) {
		*zlog.Workload = zlog.WithResults(gridtypes.Result{
			State: gridtypes.StateOk,
		})
	}
	wlVM, err := NewVMFromWorkloads(wl.Workload, &d)
	wlVM.Match(vm)
	assert.NoError(t, err)
	assert.Equal(t, *vm, wlVM)
}

func TestDestroyedIP(t *testing.T) {
	vm := vmObj().WithNetworkName("network")
	wls := vm.GenerateVMWorkload()
	for i, j := 0, len(wls)-1; i < j; i, j = i+1, j-1 {
		wls[i], wls[j] = wls[j], wls[i]
	}
	d := NewDeployment(11)
	d.Workloads = wls
	wl, err := d.Get(gridtypes.Name(vm.Name))
	assert.NoError(t, err)
	dataI, err := wl.WorkloadData()
	assert.NoError(t, err)
	pubIP := dataI.(*zos.ZMachine).Network.PublicIP
	assert.NotEmpty(t, pubIP)
	ipWl, err := d.Get(pubIP)
	assert.NoError(t, err)
	pubIPRes, err := json.Marshal(zos.PublicIPResult{
		IP:      gridtypes.MustParseIPNet("189.0.0.12/24"),
		IPv6:    gridtypes.MustParseIPNet("::/64"),
		Gateway: nil,
	})
	assert.NoError(t, err)
	*ipWl.Workload = ipWl.WithResults(gridtypes.Result{
		State: gridtypes.StateDeleted,
		Data:  json.RawMessage(pubIPRes),
	})
	vmRes, err := json.Marshal(zos.ZMachineResult{
		IP:    "",
		ID:    "",
		YggIP: "::/64",
	})
	assert.NoError(t, err)
	*wl.Workload = wl.Workload.WithResults(gridtypes.Result{
		State: gridtypes.StateOk,
		Data:  vmRes,
	})
	for _, zlog := range d.ByType(zos.ZLogsType) {
		*zlog.Workload = zlog.WithResults(gridtypes.Result{
			State: gridtypes.StateOk,
		})
	}
	wlVM, err := NewVMFromWorkloads(wl.Workload, &d)
	wlVM.Match(vm)
	vm.ComputedIP = ""
	vm.ComputedIP6 = ""
	vm.PublicIP = false
	vm.PublicIP6 = false
	assert.NoError(t, err)
	assert.Equal(t, *vm, wlVM)
}
