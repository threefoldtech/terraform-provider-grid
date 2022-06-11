package workloads

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/pkg/errors"
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

	NetworkName string
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
func NewVMFromWorkloads(wl *gridtypes.Workload, dl *gridtypes.Deployment) (VM, error) {
	dataI, err := wl.WorkloadData()
	if err != nil {
		return VM{}, errors.Wrap(err, "failed to get workload data")
	}
	// TODO: check ok?
	data := dataI.(*zos.ZMachine)
	var result zos.ZMachineResult

	if err := json.Unmarshal(wl.Result.Data, &result); err != nil {
		return VM{}, errors.Wrap(err, "failed to get vm result")
	}

	pubip := pubIP(dl, data.Network.PublicIP)

	return VM{
		Name:        wl.Name.String(),
		Description: wl.Description,
		Flist:       data.FList,
		// FIXME how to handle checksums
		FlistChecksum: "",
		PublicIP:      !pubip.IP.Nil(),
		ComputedIP:    pubip.IP.String(),
		PublicIP6:     !pubip.IPv6.Nil(),
		ComputedIP6:   pubip.IPv6.String(),
		Planetary:     result.YggIP != "",
		Corex:         data.Corex,
		YggIP:         result.YggIP,
		// FIXME check serialization
		IP:          data.Network.Interfaces[0].IP.String(),
		Cpu:         int(data.ComputeCapacity.CPU),
		Memory:      int(data.ComputeCapacity.Memory / gridtypes.Megabyte),
		RootfsSize:  int(data.Size / gridtypes.Megabyte),
		Entrypoint:  data.Entrypoint,
		Mounts:      mounts(data.Mounts),
		Zlogs:       zlogs(dl, wl.Name.String()),
		EnvVars:     data.Env,
		NetworkName: string(data.Network.Interfaces[0].Network),
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

func zlogs(dl *gridtypes.Deployment, name string) []Zlog {
	var res []Zlog
	for _, wl := range dl.ByType(zos.ZLogsType) {
		if !wl.Result.State.IsOkay() {
			continue
		}
		dataI, err := wl.WorkloadData()
		if err != nil {
			continue
		}
		data := dataI.(*zos.ZLogs)
		res = append(res, Zlog{
			Output: data.Output,
		})
	}
	return res
}

func pubIP(dl *gridtypes.Deployment, name gridtypes.Name) zos.PublicIPResult {

	pubIPWl, err := dl.Get(name)
	if err != nil || !pubIPWl.Workload.Result.State.IsOkay() {
		pubIPWl = nil
	}
	var pubIPResult zos.PublicIPResult

	_ = json.Unmarshal(pubIPWl.Result.Data, &pubIPResult)
	return pubIPResult
}

func (v *VM) GenerateVMWorkload() []gridtypes.Workload {
	workloads := make([]gridtypes.Workload, 0)
	publicIPName := ""
	if v.PublicIP || v.PublicIP6 {
		publicIPName = fmt.Sprintf("%sip", v.Name)
		workloads = append(workloads, constructPublicIPWorkload(publicIPName, v.PublicIP, v.PublicIP6))
	}
	mounts := make([]zos.MachineMount, 0)
	for _, mount := range v.Mounts {
		mounts = append(mounts, zos.MachineMount{Name: gridtypes.Name(mount.DiskName), Mountpoint: mount.MountPoint})
	}
	for _, zlog := range v.Zlogs {
		workloads = append(workloads, zlog.GenerateWorkload(v.Name))
	}
	workload := gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name(v.Name),
		Type:    zos.ZMachineType,
		Data: gridtypes.MustMarshal(zos.ZMachine{
			FList: v.Flist,
			Network: zos.MachineNetwork{
				Interfaces: []zos.MachineInterface{
					{
						Network: gridtypes.Name(v.NetworkName),
						IP:      net.ParseIP(v.IP),
					},
				},
				PublicIP:  gridtypes.Name(publicIPName),
				Planetary: v.Planetary,
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    uint8(v.Cpu),
				Memory: gridtypes.Unit(uint(v.Memory)) * gridtypes.Megabyte,
			},
			Size:       gridtypes.Unit(v.RootfsSize) * gridtypes.Megabyte,
			Entrypoint: v.Entrypoint,
			Corex:      v.Corex,
			Mounts:     mounts,
			Env:        v.EnvVars,
		}),
		Description: v.Description,
	}
	workloads = append(workloads, workload)

	return workloads
}

func (zlog *Zlog) GenerateWorkload(zmachine string) gridtypes.Workload {
	url := []byte(zlog.Output)
	urlHash := md5.Sum([]byte(url))
	return gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name(hex.EncodeToString(urlHash[:])),
		Type:    zos.ZLogsType,
		Data: gridtypes.MustMarshal(zos.ZLogs{
			ZMachine: gridtypes.Name(zmachine),
			Output:   zlog.Output,
		}),
	}
}

func (vm *VM) Dictify() map[string]interface{} {
	envVars := make(map[string]interface{})
	for key, value := range vm.EnvVars {
		envVars[key] = value
	}
	mounts := make([]map[string]interface{}, 0)
	for _, mountPoint := range vm.Mounts {
		mount := map[string]interface{}{
			"disk_name": mountPoint.DiskName, "mount_point": mountPoint.MountPoint,
		}
		mounts = append(mounts, mount)
	}
	zlogs := make([]string, 0)
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
	res["computedip"] = vm.ComputedIP
	res["computedip6"] = vm.ComputedIP6
	res["ygg_ip"] = vm.YggIP
	res["ip"] = vm.IP
	res["mounts"] = mounts
	res["cpu"] = vm.Cpu
	res["memory"] = vm.Memory
	res["rootfs_size"] = vm.RootfsSize
	res["env_vars"] = envVars
	res["entrypoint"] = vm.Entrypoint
	res["zlogs"] = zlogs
	return res
}
func (v *VM) Validate() error {
	if v.FlistChecksum != "" {
		checksum, err := getFlistChecksum(flistChecksumURL(v.Flist))
		if err != nil {
			return errors.Wrap(err, "failed to get flist checksum")
		}
		if v.FlistChecksum != checksum {
			return fmt.Errorf(
				"passed checksum %s of %s doesn't match %s returned from %s",
				v.FlistChecksum,
				v.Name,
				checksum,
				flistChecksumURL(v.Flist),
			)
		}
	}
	return nil
}
func (v *VM) WithNetworkName(name string) *VM {
	v.NetworkName = name
	return v
}

func constructPublicIPWorkload(workloadName string, ipv4 bool, ipv6 bool) gridtypes.Workload {
	return gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name(workloadName),
		Type:    zos.PublicIPType,
		Data: gridtypes.MustMarshal(zos.PublicIP{
			V4: ipv4,
			V6: ipv6,
		}),
	}
}

func flistChecksumURL(url string) string {
	return fmt.Sprintf("%s.md5", url)
}
func getFlistChecksum(url string) (string, error) {
	response, err := http.Get(flistChecksumURL(url))
	if err != nil {
		return "", err
	}
	hash, err := ioutil.ReadAll(response.Body)
	return strings.TrimSpace(string(hash)), err
}
