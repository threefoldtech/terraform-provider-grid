// Package workloads includes workloads types (vm, zdb, qsfs, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"regexp"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// ErrDuplicateName error for duplicate names
var ErrDuplicateName = errors.New("node names are not unique")

// K8sNodeData kubernetes data
type K8sNodeData struct {
	Name          string
	Node          uint32
	DiskSize      int
	PublicIP      bool
	PublicIP6     bool
	Planetary     bool
	Flist         string
	FlistChecksum string
	ComputedIP    string
	ComputedIP6   string
	YggIP         string
	IP            string
	CPU           int
	Memory        int
}

type K8sCluster struct {
	Master      *K8sNodeData
	Workers     []K8sNodeData
	Token       string
	SSHKey      string
	NetworkName string
}

func NewK8sNodeData(m map[string]interface{}) K8sNodeData {
	return K8sNodeData{
		Name:          m["name"].(string),
		Node:          uint32(m["node"].(int)),
		DiskSize:      m["disk_size"].(int),
		PublicIP:      m["publicip"].(bool),
		PublicIP6:     m["publicip6"].(bool),
		Planetary:     m["planetary"].(bool),
		Flist:         m["flist"].(string),
		FlistChecksum: m["flist_checksum"].(string),
		ComputedIP:    m["computedip"].(string),
		ComputedIP6:   m["computedip6"].(string),
		YggIP:         m["ygg_ip"].(string),
		IP:            m["ip"].(string),
		CPU:           m["cpu"].(int),
		Memory:        m["memory"].(int),
	}
}

func NewK8sNodeDataFromWorkload(w gridtypes.Workload, nodeID uint32, diskSize int, computedIP string, computedIP6 string) (K8sNodeData, error) {
	var k K8sNodeData
	data, err := w.WorkloadData()
	if err != nil {
		return k, err
	}
	d := data.(*zos.ZMachine)
	var result zos.ZMachineResult

	if !reflect.DeepEqual(w.Result, gridtypes.Result{}) {
		err = w.Result.Unmarshal(&result)
		if err != nil {
			return k, err
		}
	}

	flistCheckSum, err := GetFlistChecksum(d.FList)
	if err != nil {
		return k, err
	}

	k = K8sNodeData{
		Name:          string(w.Name),
		Node:          nodeID,
		DiskSize:      diskSize,
		PublicIP:      computedIP != "",
		PublicIP6:     computedIP6 != "",
		Planetary:     result.YggIP != "",
		Flist:         d.FList,
		FlistChecksum: flistCheckSum,
		ComputedIP:    computedIP,
		ComputedIP6:   computedIP6,
		YggIP:         result.YggIP,
		IP:            d.Network.Interfaces[0].IP.String(),
		CPU:           int(d.ComputeCapacity.CPU),
		Memory:        int(d.ComputeCapacity.Memory / gridtypes.Megabyte),
	}
	return k, nil
}

func (k *K8sNodeData) Dictify() map[string]interface{} {
	res := make(map[string]interface{})
	res["name"] = k.Name
	res["node"] = int(k.Node)
	res["disk_size"] = k.DiskSize
	res["publicip"] = k.PublicIP
	res["publicip6"] = k.PublicIP6
	res["planetary"] = k.Planetary
	res["flist"] = k.Flist
	res["flist_checksum"] = k.FlistChecksum
	res["computedip"] = k.ComputedIP
	res["computedip6"] = k.ComputedIP6
	res["ygg_ip"] = k.YggIP
	res["ip"] = k.IP
	res["cpu"] = k.CPU
	res["memory"] = k.Memory
	return res
}

func (k *K8sNodeData) GenerateK8sWorkload(cluster *K8sCluster, masterIP string) []gridtypes.Workload {
	diskName := fmt.Sprintf("%sdisk", k.Name)
	K8sWorkloads := make([]gridtypes.Workload, 0)
	diskWorkload := gridtypes.Workload{
		Name:        gridtypes.Name(diskName),
		Version:     0,
		Type:        zos.ZMountType,
		Description: "",
		Data: gridtypes.MustMarshal(zos.ZMount{
			Size: gridtypes.Unit(k.DiskSize) * gridtypes.Gigabyte,
		}),
	}
	K8sWorkloads = append(K8sWorkloads, diskWorkload)
	publicIPName := ""
	if k.PublicIP || k.PublicIP6 {
		publicIPName = fmt.Sprintf("%sip", k.Name)
		K8sWorkloads = append(K8sWorkloads, ConstructPublicIPWorkload(publicIPName, k.PublicIP, k.PublicIP6))
	}
	envVars := map[string]string{
		"SSH_KEY":           cluster.SSHKey,
		"K3S_TOKEN":         cluster.Token,
		"K3S_DATA_DIR":      "/mydisk",
		"K3S_FLANNEL_IFACE": "eth0",
		"K3S_NODE_NAME":     k.Name,
		"K3S_URL":           "",
	}
	if masterIP != "" {
		envVars["K3S_URL"] = fmt.Sprintf("https://%s:6443", masterIP)
	}
	workload := gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name(k.Name),
		Type:    zos.ZMachineType,
		Data: gridtypes.MustMarshal(zos.ZMachine{
			FList: k.Flist,
			Network: zos.MachineNetwork{
				Interfaces: []zos.MachineInterface{
					{
						Network: gridtypes.Name(cluster.NetworkName),
						IP:      net.ParseIP(k.IP),
					},
				},
				PublicIP:  gridtypes.Name(publicIPName),
				Planetary: k.Planetary,
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    uint8(k.CPU),
				Memory: gridtypes.Unit(uint(k.Memory)) * gridtypes.Megabyte,
			},
			Entrypoint: "/sbin/zinit init",
			Mounts: []zos.MachineMount{
				{Name: gridtypes.Name(diskName), Mountpoint: "/mydisk"},
			},
			Env: envVars,
		}),
	}
	K8sWorkloads = append(K8sWorkloads, workload)

	return K8sWorkloads
}

func (k *K8sCluster) ValidateToken(ctx context.Context) error {
	if k.Token == "" {
		return errors.New("empty token is now allowed")
	}

	is_alphanumeric := regexp.MustCompile(`^[a-zA-Z0-9]*$`).MatchString(k.Token)
	if !is_alphanumeric {
		return errors.New("token should be alphanumeric")
	}

	return nil
}

func (k *K8sCluster) ValidateNames(ctx context.Context) error {

	names := make(map[string]bool)
	names[k.Master.Name] = true
	for _, w := range k.Workers {
		if _, ok := names[w.Name]; ok {
			return fmt.Errorf("k8s workers and master must have unique names: %s occurred more than once", w.Name)
		}
		names[w.Name] = true
	}
	return nil
}
