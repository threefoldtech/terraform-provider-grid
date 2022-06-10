package workloads

import (
	"fmt"
	"net"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

type VM struct {
	Name          string
	Flist         string
	FlistChecksum string
	PublicIP      bool
	PublicIP6     bool
	Planetary     bool
	Corex         bool
	ComputedIP    string
	ComputedIP6   string
	YggIP         string
	IP            string
	Description   string
	Cpu           int
	Memory        int
	RootfsSize    int
	Entrypoint    string
	Mounts        []Mount
	Zlogs         []Zlog
	EnvVars       map[string]string
}

type Mount struct {
	DiskName   string
	MountPoint string
}

type Zlog struct {
	Output string
}

func NewVMFromSchema(vm map[string]interface{}) VM {
	mounts := make([]Mount, 0)
	mount_points := vm["mounts"].([]interface{})
	for _, mount_point := range mount_points {
		point := mount_point.(map[string]interface{})
		mount := Mount{DiskName: point["disk_name"].(string), MountPoint: point["mount_point"].(string)}
		mounts = append(mounts, mount)
	}
	envs := vm["env_vars"].(map[string]interface{})
	envVars := make(map[string]string)

	for k, v := range envs {
		envVars[k] = v.(string)
	}
	zlogs := make([]Zlog, 0)
	for _, v := range vm["zlogs"].([]interface{}) {
		zlogs = append(zlogs, Zlog{v.(string)})
	}

	return VM{
		Name:          vm["name"].(string),
		PublicIP:      vm["publicip"].(bool),
		PublicIP6:     vm["publicip6"].(bool),
		Flist:         vm["flist"].(string),
		FlistChecksum: vm["flist_checksum"].(string),
		ComputedIP:    vm["computedip"].(string),
		ComputedIP6:   vm["computedip6"].(string),
		YggIP:         vm["ygg_ip"].(string),
		Planetary:     vm["planetary"].(bool),
		IP:            vm["ip"].(string),
		Cpu:           vm["cpu"].(int),
		Memory:        vm["memory"].(int),
		RootfsSize:    vm["rootfs_size"].(int),
		Entrypoint:    vm["entrypoint"].(string),
		Mounts:        mounts,
		EnvVars:       envVars,
		Corex:         vm["corex"].(bool),
		Description:   vm["description"].(string),
		Zlogs:         zlogs,
	}
}
func (vm *VM) GenerateVMWorkload(deployer *DeploymentDeployer) []gridtypes.Workload {
	workloads := make([]gridtypes.Workload, 0)
	publicIPName := ""
	if vm.PublicIP || vm.PublicIP6 {
		publicIPName = fmt.Sprintf("%sip", vm.Name)
		workloads = append(workloads, constructPublicIPWorkload(publicIPName, vm.PublicIP, vm.PublicIP6))
	}
	mounts := make([]zos.MachineMount, 0)
	for _, mount := range vm.Mounts {
		mounts = append(mounts, zos.MachineMount{Name: gridtypes.Name(mount.DiskName), Mountpoint: mount.MountPoint})
	}
	workload := gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name(vm.Name),
		Type:    zos.ZMachineType,
		Data: gridtypes.MustMarshal(zos.ZMachine{
			FList: vm.Flist,
			Network: zos.MachineNetwork{
				Interfaces: []zos.MachineInterface{
					{
						Network: gridtypes.Name(deployer.NetworkName),
						IP:      net.ParseIP(vm.IP),
					},
				},
				PublicIP:  gridtypes.Name(publicIPName),
				Planetary: vm.Planetary,
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    uint8(vm.Cpu),
				Memory: gridtypes.Unit(uint(vm.Memory)) * gridtypes.Megabyte,
			},
			Size:       gridtypes.Unit(vm.RootfsSize) * gridtypes.Megabyte,
			Entrypoint: vm.Entrypoint,
			Corex:      vm.Corex,
			Mounts:     mounts,
			Env:        vm.EnvVars,
		}),
		Description: vm.Description,
	}
	workloads = append(workloads, workload)

	return workloads
}
