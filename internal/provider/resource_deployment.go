package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	substrate "github.com/threefoldtech/substrate-client"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

const (
	Version = 0
)

func resourceDeployment() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Deployment resource (zdbs + vms + disks + qsfs).",

		CreateContext: resourceDeploymentCreate,
		ReadContext:   resourceDeploymentRead,
		UpdateContext: resourceDeploymentUpdate,
		DeleteContext: resourceDeploymentDelete,

		Schema: map[string]*schema.Schema{
			"node": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Node id to place the deployment on",
			},
			"ip_range": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "IP range of the node (e.g. 10.1.2.0/24)",
			},
			"network_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Network to use for Zmachines",
			},
			"disks": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "the disk name, used to reference it in zmachine mounts",
						},
						"size": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "the disk size in GBs",
						},
						"description": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
					},
				},
			},
			"zdbs": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"password": {
							Type:     schema.TypeString,
							Required: true,
						},
						"public": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Makes it read-only if password is set, writable if no password set",
						},
						"size": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Size of the zdb in GBs",
						},
						"description": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"mode": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Mode of the zdb, user or seq",
						},
						"ips": {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Computed:    true,
							Description: "IPs of the zdb",
						},
						"namespace": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Namespace of the zdb",
						},
						"port": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Port of the zdb",
						},
					},
				},
			},
			"vms": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"flist": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "e.g. https://hub.grid.tf/omar0.3bot/omarelawady-ubuntu-20.04.flist",
						},
						"publicip": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "true to enable public ip reservation",
						},
						"computedip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The reserved public ip",
						},
						"ip": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "The private wg IP of the Zmachine",
						},
						"cpu": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     1,
							Description: "Number of VCPUs",
						},
						"description": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"memory": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Memory size",
						},
						"rootfs_size": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Rootfs size in MB",
						},
						"entrypoint": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "command to execute as the Zmachine init",
						},
						"mounts": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"disk_name": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Name of QSFS or Disk to mount",
									},
									"mount_point": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Directory to mount the disk on inside the Zmachine",
									},
								},
							},
							Description: "Zmachine mounts, can reference QSFSs and Disks",
						},
						"env_vars": {
							Type:        schema.TypeMap,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "Environment variables to pass to the zmachine",
						},
						"planetary": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Enable Yggdrasil allocation",
						},
						"ygg_ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Allocated Yggdrasil IP",
						},
					},
				},
			},
			"qsfs": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"description": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"cache": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The size of the fuse mountpoint on the node in MBs (holds qsfs local data before pushing)",
						},
						"minimal_shards": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The minimum amount of shards which are needed to recover the original data.",
						},
						"expected_shards": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The amount of shards which are generated when the data is encoded. Essentially, this is the amount of shards which is needed to be able to recover the data, and some disposable shards which could be lost. The amount of disposable shards can be calculated as expected_shards - minimal_shards.",
						},
						"redundant_groups": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The amount of groups which one should be able to loose while still being able to recover the original data.",
						},
						"redundant_nodes": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The amount of nodes that can be lost in every group while still being able to recover the original data.",
						},
						"max_zdb_data_dir_size": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Maximum size of the data dir in MiB, if this is set and the sum of the file sizes in the data dir gets higher than this value, the least used, already encoded file will be removed.",
						},
						"encryption_algorithm": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "AES",
							Description: "configuration to use for the encryption stage. Currently only AES is supported.",
						},
						"encryption_key": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "64 long hex encoded encryption key (e.g. 0000000000000000000000000000000000000000000000000000000000000000)",
						},
						"compression_algorithm": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "snappy",
							Description: "configuration to use for the compression stage. Currently only snappy is supported",
						},
						"metadata": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							MinItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:        schema.TypeString,
										Optional:    true,
										Default:     "zdb",
										Description: "configuration for the metadata store to use, currently only zdb is supported",
									},
									"prefix": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Data stored on the remote metadata is prefixed with",
									},
									"encryption_algorithm": {
										Type:        schema.TypeString,
										Optional:    true,
										Default:     "AES",
										Description: "configuration to use for the encryption stage. Currently only AES is supported.",
									},
									"encryption_key": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "64 long hex encoded encryption key (e.g. 0000000000000000000000000000000000000000000000000000000000000000)",
									},
									"backends": {
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"address": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Address of backend zdb (e.g. [300:a582:c60c:df75:f6da:8a92:d5ed:71ad]:9900 or 60.60.60.60:9900)",
												},
												"namespace": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "ZDB namespace",
												},
												"password": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Namespace password",
												},
											},
										},
									}},
							},
						},
						"groups": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "The backend groups to write the data to.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"backends": {
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"address": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Address of backend zdb (e.g. [300:a582:c60c:df75:f6da:8a92:d5ed:71ad]:9900 or 60.60.60.60:9900)",
												},
												"namespace": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "ZDB namespace",
												},
												"password": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Namespace password",
												},
											},
										},
									},
								},
							},
						},
						"metrics_endpoint": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "QSFS exposed metrics",
						},
					},
				},
			},
		},
	}
}

type Disk struct {
	Name        string
	Size        int
	Description string
}

type ZDB struct {
	Name        string
	Password    string
	Public      bool
	Size        int
	Description string
	Mode        string
	IPs         []string
	Port        uint32
	Namespace   string
}

type VM struct {
	Name        string
	Flist       string
	PublicIP    bool
	Planetary   bool
	ComputedIP  string
	YggIP       string
	IP          string
	Description string
	Cpu         int
	Memory      int
	RootfsSize  int
	Entrypoint  string
	Mounts      []Mount
	EnvVars     map[string]string
}
type Mount struct {
	DiskName   string
	MountPoint string
}
type DeploymentDeployer struct {
	Id           string
	Node         uint32
	Disks        []Disk
	ZDBs         []ZDB
	VMs          []VM
	QSFSs        []QSFS
	IPRange      *gridtypes.IPNet
	UsedIPs      []string
	NetworkName  string
	NodesIPRange map[uint32]gridtypes.IPNet
	APIClient    *apiClient
}

func getFreeIP(ipRange gridtypes.IPNet, usedIPs []string) (string, error) {
	i := 2
	l := len(ipRange.IP)
	for i < 255 {
		ip := ipNet(ipRange.IP[l-4], ipRange.IP[l-3], ipRange.IP[l-2], byte(i), 32)
		ipStr := fmt.Sprintf("%d.%d.%d.%d", ip.IP[l-4], ip.IP[l-3], ip.IP[l-2], ip.IP[l-1])
		log.Printf("ip string: %s\n", ipStr)
		if !isInStr(usedIPs, ipStr) {
			return ipStr, nil
		}
		i += 1
	}
	return "", errors.New("all ips are used")
}

func constructPublicIPWorkload(workloadName string) gridtypes.Workload {
	return gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name(workloadName),
		Type:    zos.PublicIPType,
		Data: gridtypes.MustMarshal(zos.PublicIP{
			V4: true,
		}),
	}
}

type PubIPData struct {
	IP      string `json:"ip"`
	Gateway string `json:"gateway"`
}

func GetVMData(vm map[string]interface{}) VM {
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
	return VM{
		Name:        vm["name"].(string),
		PublicIP:    vm["publicip"].(bool),
		Flist:       vm["flist"].(string),
		ComputedIP:  vm["computedip"].(string),
		YggIP:       vm["ygg_ip"].(string),
		Planetary:   vm["planetary"].(bool),
		IP:          vm["ip"].(string),
		Cpu:         vm["cpu"].(int),
		Memory:      vm["memory"].(int),
		RootfsSize:  vm["rootfs_size"].(int),
		Entrypoint:  vm["entrypoint"].(string),
		Mounts:      mounts,
		EnvVars:     envVars,
		Description: vm["description"].(string),
	}
}

func GetDiskData(disk map[string]interface{}) Disk {
	return Disk{
		Name:        disk["name"].(string),
		Size:        disk["size"].(int),
		Description: disk["description"].(string),
	}
}
func GetZdbData(zdb map[string]interface{}) ZDB {
	ipsIf := zdb["ips"].([]interface{})
	ips := make([]string, len(ipsIf))
	for idx, ip := range ipsIf {
		ips[idx] = ip.(string)
	}

	return ZDB{
		Name:        zdb["name"].(string),
		Size:        zdb["size"].(int),
		Description: zdb["description"].(string),
		Password:    zdb["password"].(string),
		Public:      zdb["public"].(bool),
		Mode:        zdb["mode"].(string),
		IPs:         ips,
		Port:        uint32(zdb["port"].(int)),
		Namespace:   zdb["namespace"].(string),
	}
}

func getDeploymentDeployer(d *schema.ResourceData, apiClient *apiClient) (DeploymentDeployer, error) {
	ipRangeStr := d.Get("ip_range").(string)
	var ipRange *gridtypes.IPNet
	if ipRangeStr == "" {
		ipRange = nil
	} else {
		r, err := gridtypes.ParseIPNet(ipRangeStr)
		if err != nil {
			return DeploymentDeployer{}, err
		}
		ipRange = &r
	}
	usedIPs := make([]string, 0)

	disks := make([]Disk, 0)
	for _, disk := range d.Get("disks").([]interface{}) {
		data := GetDiskData(disk.(map[string]interface{}))
		disks = append(disks, data)
	}

	zdbs := make([]ZDB, 0)
	for _, zdb := range d.Get("zdbs").([]interface{}) {
		data := GetZdbData(zdb.(map[string]interface{}))
		zdbs = append(zdbs, data)
	}

	vms := make([]VM, 0)
	for _, vm := range d.Get("vms").([]interface{}) {
		data := GetVMData(vm.(map[string]interface{}))
		vms = append(vms, data)
		if data.IP != "" {
			usedIPs = append(usedIPs, data.IP)
		}
	}
	qsfs := make([]QSFS, 0)
	for _, q := range d.Get("qsfs").([]interface{}) {
		data := NewQSFSFromSchema(q.(map[string]interface{}))
		qsfs = append(qsfs, data)
	}
	deploymentDeployer := DeploymentDeployer{
		Id:          d.Id(),
		Node:        uint32(d.Get("node").(int)),
		Disks:       disks,
		VMs:         vms,
		QSFSs:       qsfs,
		ZDBs:        zdbs,
		IPRange:     ipRange,
		UsedIPs:     usedIPs,
		NetworkName: d.Get("network_name").(string),
		APIClient:   apiClient,
	}
	return deploymentDeployer, nil
}

func (d *DeploymentDeployer) assignNodesIPs() error {
	for idx, vm := range d.VMs {
		if vm.IP != "" && d.IPRange.Contains(net.ParseIP(vm.IP)) {
			continue
		}
		ip, err := getFreeIP(*d.IPRange, d.UsedIPs)
		if err != nil {
			return errors.Wrap(err, "failed to find free ip for VM")
		}
		d.VMs[idx].IP = ip
		d.UsedIPs = append(d.UsedIPs, ip)
	}
	return nil
}
func (d *Disk) GenerateDiskWorkload() gridtypes.Workload {
	workload := gridtypes.Workload{
		Name:        gridtypes.Name(d.Name),
		Version:     0,
		Type:        zos.ZMountType,
		Description: d.Description,
		Data: gridtypes.MustMarshal(zos.ZMount{
			Size: gridtypes.Unit(d.Size) * gridtypes.Gigabyte,
		}),
	}

	return workload
}
func (z *ZDB) GenerateZDBWorkload() gridtypes.Workload {
	workload := gridtypes.Workload{
		Name:        gridtypes.Name(z.Name),
		Type:        zos.ZDBType,
		Description: z.Description,
		Version:     Version,
		Data: gridtypes.MustMarshal(zos.ZDB{
			Size:     gridtypes.Unit(z.Size) * gridtypes.Gigabyte,
			Mode:     zos.ZDBMode(z.Mode),
			Password: z.Password,
			Public:   z.Public,
		}),
	}
	return workload
}
func (vm *VM) GenerateVMWorkload(deployer *DeploymentDeployer) []gridtypes.Workload {
	workloads := make([]gridtypes.Workload, 0)
	publicIPName := ""
	if vm.PublicIP {
		publicIPName = fmt.Sprintf("%sip", vm.Name)
		workloads = append(workloads, constructPublicIPWorkload(publicIPName))
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
			Mounts:     mounts,
			Env:        vm.EnvVars,
		}),
	}
	workloads = append(workloads, workload)

	return workloads
}
func (d *DeploymentDeployer) GenerateVersionlessDeployments(ctx context.Context) (map[uint32]gridtypes.Deployment, error) {
	deployments := make(map[uint32]gridtypes.Deployment)
	err := d.assignNodesIPs()
	if err != nil {
		return nil, errors.Wrap(err, "failed to assign node ips")
	}
	workloads := make([]gridtypes.Workload, 0)
	for _, disk := range d.Disks {
		workload := disk.GenerateDiskWorkload()
		workloads = append(workloads, workload)
	}
	for _, zdb := range d.ZDBs {
		workload := zdb.GenerateZDBWorkload()
		workloads = append(workloads, workload)
	}
	for _, vm := range d.VMs {
		vmWorkloads := vm.GenerateVMWorkload(d)
		workloads = append(workloads, vmWorkloads...)
	}

	for idx, q := range d.QSFSs {
		qsfsWorkload, err := q.GenerateWorkload(d)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate qsfs %d", idx)
		}
		workloads = append(workloads, qsfsWorkload)
	}

	deployment := gridtypes.Deployment{
		Version: 0,
		TwinID:  uint32(d.APIClient.twin_id), //LocalTwin,
		// this contract id must match the one on substrate
		Workloads: workloads,
		SignatureRequirement: gridtypes.SignatureRequirement{
			WeightRequired: 1,
			Requests: []gridtypes.SignatureRequest{
				{
					TwinID: d.APIClient.twin_id,
					Weight: 1,
				},
			},
		},
	}
	deployments[d.Node] = deployment
	return deployments, nil
}
func (d *DeploymentDeployer) getNodeClient(nodeID uint32) (*client.NodeClient, error) {
	nodeInfo, err := d.APIClient.sub.GetNode(nodeID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get node")
	}

	cl := client.NewNodeClient(uint32(nodeInfo.TwinID), d.APIClient.rmb)
	return cl, nil
}
func (d *DeploymentDeployer) GetOldDeployments(ctx context.Context) (map[uint32]uint64, error) {
	deployments := make(map[uint32]uint64)
	if d.Id != "" {

		deploymentID, err := strconv.ParseUint(d.Id, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't parse deployment id %s", d.Id)
		}
		deployments[d.Node] = deploymentID
	}

	return deployments, nil
}
func (d *DeploymentDeployer) updateState(ctx context.Context, currentDeploymentIDs map[uint32]uint64) error {
	log.Printf("current deployments\n")
	currentDeployments, err := getDeploymentObjects(ctx, currentDeploymentIDs, d)
	if err != nil {
		return errors.Wrap(err, "failed to get deployments to update local state")
	}
	printDeployments(currentDeployments)
	publicIPs := make(map[string]string)
	yggIPs := make(map[string]string)
	privateIPs := make(map[string]string)
	zdbIPs := make(map[string][]string)
	zdbPort := make(map[string]uint)
	zdbNamespace := make(map[string]string)
	workloads := make(map[string]*gridtypes.Workload)

	for _, dl := range currentDeployments {
		for _, w := range dl.Workloads {
			if w.Type == zos.PublicIPType {
				d := PubIPData{}
				if err := json.Unmarshal(w.Result.Data, &d); err != nil {
					log.Printf("error unmarshalling json: %s\n", err)
					continue
				}
				publicIPs[string(w.Name)] = d.IP
			} else if w.Type == zos.ZMachineType {
				d, err := w.WorkloadData()
				if err != nil {
					log.Printf("error loading machine data: %s\n", err)
					continue
				}
				res := zos.ZMachineResult{}
				if err := json.Unmarshal(w.Result.Data, &res); err != nil {
					log.Printf("error unmarshalling json: %s\n", err)
					continue
				}
				privateIPs[string(w.Name)] = d.(*zos.ZMachine).Network.Interfaces[0].IP.String()
				yggIPs[string(w.Name)] = res.YggIP
			} else if w.Type == zos.ZDBType {
				d := zos.ZDBResult{}
				if err := json.Unmarshal(w.Result.Data, &d); err != nil {
					log.Printf("error unmarshalling json: %s\n", err)
					continue
				}
				zdbIPs[string(w.Name)] = d.IPs
				zdbPort[string(w.Name)] = d.Port
				zdbNamespace[string(w.Name)] = d.Namespace
			} else if w.Type == zos.QuantumSafeFSType {
				workloads[string(w.Name)] = &w
			}
		}
	}
	for idx, vm := range d.VMs {
		vmIPName := fmt.Sprintf("%sip", vm.Name)
		if ip, ok := publicIPs[vmIPName]; ok {
			d.VMs[idx].ComputedIP = ip
			d.VMs[idx].PublicIP = true
		} else {
			d.VMs[idx].ComputedIP = ""
			d.VMs[idx].PublicIP = false
		}
		private, ok := privateIPs[string(vm.Name)]
		if ok {
			d.VMs[idx].IP = private
		} else {
			d.VMs[idx].IP = ""
		}
		ygg, ok := yggIPs[string(vm.Name)]
		if ok {
			d.VMs[idx].YggIP = ygg
			d.VMs[idx].Planetary = true
		} else {
			d.VMs[idx].YggIP = ""
			d.VMs[idx].Planetary = false
		}
	}
	for idx, zdb := range d.ZDBs {
		if ips, ok := zdbIPs[zdb.Name]; ok {
			d.ZDBs[idx].IPs = ips
			d.ZDBs[idx].Port = uint32(zdbPort[zdb.Name])
			d.ZDBs[idx].Namespace = zdbNamespace[zdb.Name]
		} else {
			d.ZDBs[idx].IPs = make([]string, 0)
			d.ZDBs[idx].Port = 0
			d.ZDBs[idx].Namespace = ""
		}
	}
	for idx := range d.QSFSs {
		name := string(d.QSFSs[idx].Name)
		if err := d.QSFSs[idx].updateFromWorkload(workloads[name]); err != nil {
			log.Printf("couldn't update qsfs from workload: %s\n", err)
		}
	}
	log.Printf("Current state after updatestate %v\n", d)
	return nil
}

func (d *DeploymentDeployer) Deploy(ctx context.Context) (uint32, error) {
	newDeployments, err := d.GenerateVersionlessDeployments(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "couldn't generate deployments data")
	}
	oldDeployments, err := d.GetOldDeployments(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "couldn't get old deployments data")
	}
	currentDeployments, err := deployDeployments(ctx, oldDeployments, newDeployments, d, d.APIClient, true)
	if err := d.updateState(ctx, currentDeployments); err != nil {
		log.Printf("error updating state: %s\n", err)
	}
	return uint32(currentDeployments[d.Node]), err
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
	res := make(map[string]interface{})
	res["name"] = vm.Name
	res["description"] = vm.Description
	res["publicip"] = vm.PublicIP
	res["planetary"] = vm.Planetary
	res["flist"] = vm.Flist
	res["computedip"] = vm.ComputedIP
	res["ygg_ip"] = vm.YggIP
	res["ip"] = vm.IP
	res["mounts"] = mounts
	res["cpu"] = vm.Cpu
	res["memory"] = vm.Memory
	res["rootfs_size"] = vm.RootfsSize
	res["env_vars"] = envVars
	res["entrypoint"] = vm.Entrypoint
	return res
}
func (d *Disk) Dictify() map[string]interface{} {
	res := make(map[string]interface{})
	res["name"] = d.Name
	res["description"] = d.Description
	res["size"] = d.Size
	return res
}
func (z *ZDB) Dictify() map[string]interface{} {
	res := make(map[string]interface{})
	res["name"] = z.Name
	res["description"] = z.Description
	res["size"] = z.Size
	res["mode"] = z.Mode
	res["ips"] = z.IPs
	res["namespace"] = z.Namespace
	res["port"] = int(z.Port)
	res["password"] = z.Password
	res["public"] = z.Public
	return res
}
func (dep *DeploymentDeployer) storeState(d *schema.ResourceData) {
	vms := make([]interface{}, 0)
	for _, vm := range dep.VMs {
		vms = append(vms, vm.Dictify())
	}
	disks := make([]interface{}, 0)
	for _, d := range dep.Disks {
		disks = append(disks, d.Dictify())
	}
	zdbs := make([]interface{}, 0)
	for _, zdb := range dep.ZDBs {
		zdbs = append(zdbs, zdb.Dictify())
	}
	qsfs := make([]interface{}, 0)
	for _, q := range dep.QSFSs {
		qsfs = append(zdbs, q.Dictify())
	}
	d.Set("vms", vms)
	d.Set("zdbs", zdbs)
	d.Set("disks", disks)
	d.Set("qsfs", qsfs)
	d.Set("node", dep.Node)
	d.Set("network_name", dep.NetworkName)
	if dep.IPRange != nil {
		d.Set("ip_range", dep.IPRange.String())
	}
}
func resourceDeploymentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := validate(d)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error validating deployment"))
	}
	apiClient := meta.(*apiClient)
	if err := validateAccountMoneyForExtrinsics(apiClient); err != nil {
		return diag.FromErr(err)
	}
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := getDeploymentDeployer(d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	var diags diag.Diagnostics
	deploymentID, err := deployer.Deploy(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	deployer.storeState(d)
	d.SetId(strconv.FormatUint(uint64(deploymentID), 10))
	return diags
}

func flattenDiskData(workload gridtypes.Workload) (map[string]interface{}, error) {
	if workload.Type == zos.ZMountType {
		wl := make(map[string]interface{})
		data, err := workload.WorkloadData()
		if err != nil {
			return nil, err
		}
		wl["name"] = workload.Name
		wl["size"] = data.(*zos.ZMount).Size / gridtypes.Gigabyte
		wl["description"] = workload.Description
		return wl, nil
	}

	return nil, errors.New("The wrokload is not of type zos.ZMountType")
}
func flattenZDBData(workload gridtypes.Workload) (map[string]interface{}, error) {
	if workload.Type == zos.ZDBType {
		wl := make(map[string]interface{})
		data, err := workload.WorkloadData()
		if err != nil {
			return nil, err
		}
		var result zos.ZDBResult
		if err := workload.Result.Unmarshal(&result); err != nil {
			return nil, errors.Wrap(err, "couldn't decode zdb result")
		}
		wl["name"] = workload.Name
		wl["size"] = data.(*zos.ZDB).Size / gridtypes.Gigabyte
		wl["mode"] = data.(*zos.ZDB).Mode
		wl["ips"] = result.IPs
		wl["namespace"] = result.Namespace
		wl["port"] = result.Port
		wl["password"] = data.(*zos.ZDB).Password
		wl["public"] = data.(*zos.ZDB).Public
		wl["description"] = workload.Description
		return wl, nil
	}

	return nil, errors.New("The wrokload is not of type zos.ZDBType")
}

func flattenVMData(workload gridtypes.Workload) (map[string]interface{}, error) {
	if workload.Type == zos.ZMachineType {
		wl := make(map[string]interface{})
		workloadData, err := workload.WorkloadData()
		if err != nil {
			return nil, err
		}
		data := workloadData.(*zos.ZMachine)

		mounts := make([]map[string]interface{}, 0)
		for _, mountPoint := range data.Mounts {
			mount := map[string]interface{}{
				"disk_name": string(mountPoint.Name), "mount_point": mountPoint.Mountpoint,
			}
			mounts = append(mounts, mount)
		}
		envVars := make(map[string]interface{})
		for key, value := range data.Env {
			envVars[key] = value
		}
		machineData, err := workload.WorkloadData()
		if err != nil {
			return nil, err
		}
		var result zos.ZMachineResult
		if err := workload.Result.Unmarshal(&result); err != nil {
			return nil, errors.Wrap(err, "couldn't decode zdb result")
		}

		wl["cpu"] = data.ComputeCapacity.CPU
		wl["memory"] = uint64(data.ComputeCapacity.Memory) / uint64(gridtypes.Megabyte)
		wl["rootfs_size"] = uint64(data.Size) / uint64(gridtypes.Megabyte)
		wl["mounts"] = mounts
		wl["name"] = workload.Name
		wl["flist"] = data.FList
		wl["entrypoint"] = data.Entrypoint
		wl["description"] = workload.Description
		wl["env_vars"] = envVars
		wl["ip"] = machineData.(*zos.ZMachine).Network.Interfaces[0].IP.String()
		wl["ygg_ip"] = result.YggIP
		wl["planetary"] = result.YggIP != ""
		return wl, nil
	}

	return nil, errors.New("The wrokload is not of type zos.ZMachineType")
}

func resourceDeploymentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta valufreeIPe to retrieve your client from the provider configure method
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	cl := apiClient.rmb
	var diags diag.Diagnostics
	sub, err := substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting substrate client"))
	}
	nodeID := uint32(d.Get("node").(int))
	nodeInfo, err := sub.GetNode(nodeID)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Error reading data from remote, terraform state might be out of sync with the remote state",
			Detail:   errors.Wrap(err, "error getting node client").Error(),
		})
		return diags
	}

	node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)

	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	contractId, err := strconv.ParseUint(d.Id(), 10, 64)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Error reading data from remote, terraform state might be out of sync with the remote state",
			Detail:   errors.Wrap(err, "error parsing contract id").Error(),
		})
		return diags
	}
	subctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	deployment, err := node.DeploymentGet(subctx, contractId)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Error reading data from remote, terraform state might be out of sync with the remote state",
			Detail:   errors.Wrap(err, "error getting deployment").Error(),
		})
		return diags
	}
	qsfs := make([]map[string]interface{}, 0)
	disks := make([]map[string]interface{}, 0)
	zdbs := make([]map[string]interface{}, 0)
	vms := make([]map[string]interface{}, 0)
	publicIPs := make(map[string]string)
	for _, workload := range deployment.Workloads {
		if workload.Type == zos.ZMountType {
			flattened, err := flattenDiskData(workload)
			if err != nil {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "Error reading data from remote, terraform state might be out of sync with the remote state",
					Detail:   errors.Wrap(err, "error flattening disk").Error(),
				})
				return diags
			}
			disks = append(disks, flattened)

		}
		if workload.Type == zos.ZDBType {
			flattened, err := flattenZDBData(workload)
			if err != nil {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "Error reading data from remote, terraform state might be out of sync with the remote state",
					Detail:   errors.Wrap(err, "error flattening zdb").Error(),
				})
				return diags
			}
			zdbs = append(zdbs, flattened)

		} else if workload.Type == zos.ZMachineType {
			flattened, err := flattenVMData(workload)
			if err != nil {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "Error reading data from remote, terraform state might be out of sync with the remote state",
					Detail:   errors.Wrap(err, "error flattening vm").Error(),
				})
				return diags
			}
			vms = append(vms, flattened)
		} else if workload.Type == zos.PublicIPType {
			ipData := PubIPData{}
			if err := json.Unmarshal(workload.Result.Data, &ipData); err != nil {
				log.Printf("error unmarshalling json: %s\n", err)
				continue
			}
			publicIPs[string(workload.Name)] = ipData.IP
		} else if workload.Type == zos.QuantumSafeFSType {
			q, err := NewQSFSFromWorkload(&workload)
			if err != nil {
				log.Printf("error getting qsfs from workload: %s\n", err)
				continue
			}
			qsfs = append(qsfs, q.Dictify())
		}
	}

	for _, vm := range vms {
		vmIPName := fmt.Sprintf("%sip", vm["name"])
		if ip, ok := publicIPs[vmIPName]; ok {
			vm["computedip"] = ip
			vm["publicip"] = true
		} else {
			vm["computedip"] = ""
			vm["publicip"] = false
		}
	}

	d.Set("vms", vms)
	d.Set("disks", disks)
	d.Set("zdbs", zdbs)
	d.Set("qsfs", qsfs)
	return diags
}

func validate(d *schema.ResourceData) error {
	ipRangeStr := d.Get("ip_range").(string)
	networkName := d.Get("network_name").(string)
	vms := d.Get("vms").([]interface{})
	_, _, err := net.ParseCIDR(ipRangeStr)
	if len(vms) != 0 && err != nil {
		return errors.Wrap(err, "If you pass a vm, ip_range must be set to a valid ip range (e.g. 10.1.3.0/16)")
	}
	if len(vms) != 0 && networkName == "" {
		return errors.Wrap(err, "If you pass a vm, network_name must be non-empty")
	}

	return nil
}

func resourceDeploymentUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := validate(d)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error validating deployment"))
	}
	if d.HasChange("node") {
		return diag.FromErr(errors.New("changing node is not supported, you need to destroy the deployment and reapply it again but you will lose your old data"))
	}
	apiClient := meta.(*apiClient)
	if err := validateAccountMoneyForExtrinsics(apiClient); err != nil {
		return diag.FromErr(err)
	}
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := getDeploymentDeployer(d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	var diags diag.Diagnostics
	_, err = deployer.Deploy(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	deployer.storeState(d)
	return diags
}

func (d *DeploymentDeployer) Cancel(ctx context.Context) error {
	newDeployments := make(map[uint32]gridtypes.Deployment)
	oldDeployments, err := d.GetOldDeployments(ctx)
	if err != nil {
		return err
	}
	currentDeployments, err := deployDeployments(ctx, oldDeployments, newDeployments, d, d.APIClient, true)
	if err := d.updateState(ctx, currentDeployments); err != nil {
		log.Printf("error updating state: %s\n", err)
	}

	return err
}

func resourceDeploymentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(*apiClient)
	if err := validateAccountMoneyForExtrinsics(apiClient); err != nil {
		return diag.FromErr(err)
	}
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmbIfNeeded(rmbctx, apiClient)
	deployer, err := getDeploymentDeployer(d, apiClient)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load deployer data"))
	}

	var diags diag.Diagnostics
	err = deployer.Cancel(ctx)
	if err != nil {
		diags = diag.FromErr(err)
	}
	if err == nil {
		d.SetId("")
	} else {
		deployer.storeState(d)
	}
	return diags

}
