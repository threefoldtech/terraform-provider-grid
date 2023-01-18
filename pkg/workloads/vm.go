// Package workloads includes workloads types (vm, zdb, qsfs, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/json"
	"fmt"
	"net"
	"sort"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// VM is a virtual machine struct
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
	CPU           int
	Memory        int
	RootfsSize    int
	Entrypoint    string
	Mounts        []Mount
	Zlogs         []Zlog
	EnvVars       map[string]string

	NetworkName string
}

// Mount disks struct
type Mount struct {
	DiskName   string
	MountPoint string
}

// NewVMFromSchema generates a new vm from a map of its data
func NewVMFromSchema(vm map[string]interface{}) *VM {
	mounts := make([]Mount, 0)
	mountPoints := vm["mounts"].([]interface{})
	for _, mountPoint := range mountPoints {
		point := mountPoint.(map[string]interface{})
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

	return &VM{
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
		CPU:           vm["cpu"].(int),
		Memory:        vm["memory"].(int),
		RootfsSize:    vm["rootfs_size"].(int),
		Entrypoint:    vm["entrypoint"].(string),
		Mounts:        mounts,
		EnvVars:       envVars,
		Corex:         vm["corex"].(bool),
		Description:   vm["description"].(string),
		Zlogs:         zlogs,
		NetworkName:   vm["network_name"].(string),
	}
}

// NewVMFromWorkloads generates a new vm from given workloads and deployment
func NewVMFromWorkloads(wl *gridtypes.Workload, dl *gridtypes.Deployment) (VM, error) {
	dataI, err := wl.WorkloadData()
	if err != nil {
		return VM{}, errors.Wrap(err, "failed to get workload data")
	}
	// TODO: check ok?
	data := dataI.(*zos.ZMachine)
	var result zos.ZMachineResult

	if len(wl.Result.Data) > 0 {
		if err := json.Unmarshal(wl.Result.Data, &result); err != nil {
			return VM{}, errors.Wrap(err, "failed to get vm result")
		}
	}

	pubIP := pubIP(dl, data.Network.PublicIP)
	var pubIP4, pubIP6 = "", ""
	if !pubIP.IP.Nil() {
		pubIP4 = pubIP.IP.String()
	}
	if !pubIP.IPv6.Nil() {
		pubIP6 = pubIP.IPv6.String()
	}
	return VM{
		Name:          wl.Name.String(),
		Description:   wl.Description,
		Flist:         data.FList,
		FlistChecksum: "",
		PublicIP:      !pubIP.IP.Nil(),
		ComputedIP:    pubIP4,
		PublicIP6:     !pubIP.IPv6.Nil(),
		ComputedIP6:   pubIP6,
		Planetary:     result.YggIP != "",
		Corex:         data.Corex,
		YggIP:         result.YggIP,
		IP:            data.Network.Interfaces[0].IP.String(),
		CPU:           int(data.ComputeCapacity.CPU),
		Memory:        int(data.ComputeCapacity.Memory / gridtypes.Megabyte),
		RootfsSize:    int(data.Size / gridtypes.Megabyte),
		Entrypoint:    data.Entrypoint,
		Mounts:        mounts(data.Mounts),
		Zlogs:         zlogs(dl, wl.Name.String()),
		EnvVars:       data.Env,
		NetworkName:   string(data.Network.Interfaces[0].Network),
	}, nil
}

func mounts(mounts []zos.MachineMount) []Mount {
	var res []Mount
	for _, mount := range mounts {
		res = append(res, Mount{
			DiskName:   mount.Name.String(),
			MountPoint: mount.Mountpoint,
		})
	}
	return res
}

func pubIP(dl *gridtypes.Deployment, name gridtypes.Name) zos.PublicIPResult {

	pubIPWl, err := dl.Get(name)
	if err != nil || !pubIPWl.Workload.Result.State.IsOkay() {
		pubIPWl = nil
		return zos.PublicIPResult{}
	}
	var pubIPResult zos.PublicIPResult

	_ = json.Unmarshal(pubIPWl.Result.Data, &pubIPResult)
	return pubIPResult
}

// GenerateVMWorkload generates a vm workloads
func (vm *VM) GenerateVMWorkload() []gridtypes.Workload {
	workloads := make([]gridtypes.Workload, 0)
	publicIPName := ""
	if vm.PublicIP || vm.PublicIP6 {
		publicIPName = fmt.Sprintf("%sip", vm.Name)
		workloads = append(workloads, ConstructPublicIPWorkload(publicIPName, vm.PublicIP, vm.PublicIP6))
	}
	mounts := make([]zos.MachineMount, 0)
	for _, mount := range vm.Mounts {
		mounts = append(mounts, zos.MachineMount{Name: gridtypes.Name(mount.DiskName), Mountpoint: mount.MountPoint})
	}
	for _, zlog := range vm.Zlogs {
		workloads = append(workloads, zlog.GenerateWorkload(vm.Name))
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
						Network: gridtypes.Name(vm.NetworkName),
						IP:      net.ParseIP(vm.IP),
					},
				},
				PublicIP:  gridtypes.Name(publicIPName),
				Planetary: vm.Planetary,
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    uint8(vm.CPU),
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

// Dictify converts vm data to a map (dict)
func (vm *VM) Dictify() map[string]interface{} {
	envVars := make(map[string]interface{})
	for key, value := range vm.EnvVars {
		envVars[key] = value
	}
	mounts := make([]interface{}, 0)
	for _, mountPoint := range vm.Mounts {
		mount := map[string]interface{}{
			"disk_name": mountPoint.DiskName, "mount_point": mountPoint.MountPoint,
		}
		mounts = append(mounts, mount)
	}
	zlogs := make([]interface{}, 0)
	for _, zlog := range vm.Zlogs {
		zlogs = append(zlogs, zlog.Output)
	}
	res := make(map[string]interface{})
	res["name"] = vm.Name
	res["description"] = vm.Description
	res["publicip"] = vm.PublicIP
	res["publicip6"] = vm.PublicIP6
	res["planetary"] = vm.Planetary
	res["corex"] = vm.Corex
	res["flist"] = vm.Flist
	res["flist_checksum"] = vm.FlistChecksum
	res["computedip"] = vm.ComputedIP
	res["computedip6"] = vm.ComputedIP6
	res["ygg_ip"] = vm.YggIP
	res["ip"] = vm.IP
	res["mounts"] = mounts
	res["cpu"] = vm.CPU
	res["memory"] = vm.Memory
	res["rootfs_size"] = vm.RootfsSize
	res["env_vars"] = envVars
	res["entrypoint"] = vm.Entrypoint
	res["zlogs"] = zlogs
	res["network_name"] = vm.NetworkName
	return res
}

// Validate validates a virtual machine data
func (vm *VM) Validate() error {
	if vm.CPU < 1 || vm.CPU > 32 {
		return errors.Wrap(errors.New("Invalid CPU input"), "CPUs must be more than or equal to 1 and less than or equal to 32")
	}

	if vm.FlistChecksum != "" {
		checksum, err := GetFlistChecksum(vm.Flist)
		if err != nil {
			return errors.Wrap(err, "failed to get flist checksum")
		}
		if vm.FlistChecksum != checksum {
			return fmt.Errorf(
				"passed checksum %s of %s doesn't match %s returned from %s",
				vm.FlistChecksum,
				vm.Name,
				checksum,
				FlistChecksumURL(vm.Flist),
			)
		}
	}
	return nil
}

// WithNetworkName sets network name for vm
func (vm *VM) WithNetworkName(name string) *VM {
	vm.NetworkName = name
	return vm
}

// Match compares the vm with another given vm
func (vm *VM) Match(vm2 *VM) {
	l := len(vm2.Zlogs) + len(vm2.Mounts)
	names := make(map[string]int)
	for idx, zlog := range vm2.Zlogs {
		names[zlog.Output] = idx - l
	}
	for idx, mount := range vm2.Mounts {
		names[mount.DiskName] = idx - l
	}
	sort.Slice(vm.Zlogs, func(i, j int) bool {
		return names[vm.Zlogs[i].Output] < names[vm.Zlogs[j].Output]
	})
	sort.Slice(vm.Mounts, func(i, j int) bool {
		return names[vm.Mounts[i].DiskName] < names[vm.Mounts[j].DiskName]
	})
	vm.FlistChecksum = vm2.FlistChecksum
}
