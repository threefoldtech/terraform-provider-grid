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
	"github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/substrate"
)

const (
	Version = 0
)

func resourceDeployment() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Sample resource in the Terraform provider scaffolding.",

		CreateContext: resourceDeploymentCreate,
		ReadContext:   resourceDeploymentRead,
		UpdateContext: resourceDeploymentUpdate,
		DeleteContext: resourceDeploymentDelete,

		Schema: map[string]*schema.Schema{

			"node": {
				Description: "Node id to place deployment on",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"disks": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"size": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"description": {
							Type:     schema.TypeString,
							Required: true,
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
						"size": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"description": {
							Type:     schema.TypeString,
							Required: true,
						},
						"mode": {
							Description: "Mode of the zdb",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
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
							Type:     schema.TypeString,
							Required: true,
						},
						"publicip": {
							Description: "If you want to enable public ip or not",
							Type:        schema.TypeBool,
							Optional:    true,
						},
						"computedip": {
							Description: "The public ip",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"ip": {
							Description: "IP",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						"cpu": {
							Description: "CPU size",
							Type:        schema.TypeInt,
							Optional:    true,
						},
						"description": {
							Description: "Machine Description",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"memory": {
							Description: "Memory size",
							Type:        schema.TypeInt,
							Optional:    true,
						},
						"entrypoint": {
							Description: "VM entry point",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"mounts": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"disk_name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"mount_point": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"env_vars": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:     schema.TypeString,
										Required: true,
									},
									"value": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			"ip_range": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"network_name": {
				Type:     schema.TypeString,
				Optional: true,
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
	Size        int
	Description string
	Mode        string
}

type VM struct {
	Name        string
	Flist       string
	PublicIP    bool
	ComputedIP  string
	IP          string
	Description string
	Cpu         int
	Memory      int
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
	IPRange      gridtypes.IPNet
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
		Data:    gridtypes.MustMarshal(zos.PublicIP{}),
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
	envs := vm["env_vars"].([]interface{})
	envVars := make(map[string]string)

	for _, env := range envs {
		envVar := env.(map[string]interface{})
		envVars[envVar["key"].(string)] = envVar["value"].(string)
	}
	return VM{
		Name:        vm["name"].(string),
		PublicIP:    vm["publicip"].(bool),
		Flist:       vm["flist"].(string),
		ComputedIP:  vm["computedip"].(string),
		IP:          vm["ip"].(string),
		Cpu:         vm["cpu"].(int),
		Memory:      vm["memory"].(int),
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
	return ZDB{
		Name:        zdb["name"].(string),
		Size:        zdb["size"].(int),
		Description: zdb["description"].(string),
		Password:    zdb["password"].(string),
		Mode:        zdb["mode"].(string),
	}
}
func getDeploymentDeployer(d *schema.ResourceData, apiClient *apiClient) (DeploymentDeployer, error) {
	ipRangeStr := d.Get("ip_range").(string)
	ipRange, err := gridtypes.ParseIPNet(ipRangeStr)
	if err != nil {
		return DeploymentDeployer{}, err
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
	deploymentDeployer := DeploymentDeployer{
		Id:          d.Id(),
		Node:        uint32(d.Get("node").(int)),
		Disks:       disks,
		VMs:         vms,
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
		ip, err := getFreeIP(d.IPRange, d.UsedIPs)
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
			Size:     gridtypes.Unit(z.Size),
			Mode:     zos.ZDBMode(z.Mode),
			Password: z.Password,
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
				PublicIP: gridtypes.Name(publicIPName),
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    uint8(vm.Cpu),
				Memory: gridtypes.Unit(uint(vm.Memory)) * gridtypes.Megabyte,
			},
			Entrypoint: "/sbin/zinit init",
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
func (d *DeploymentDeployer) GetOldDeployments(ctx context.Context) (map[uint32]gridtypes.Deployment, error) {
	deployments := make(map[uint32]gridtypes.Deployment)
	if d.Id == "" {
		return deployments, nil
	}
	client, err := d.getNodeClient(d.Node)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get node client")
	}

	deploymentID, err := strconv.ParseUint(d.Id, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get node client")
	}
	log.Printf("DEPLOYMENT_ID %s", d.Id)
	deployment, err := client.DeploymentGet(ctx, deploymentID)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't fetch deployment")
	}
	deployments[d.Node] = deployment
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
	privateIPs := make(map[string]string)
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
				privateIPs[string(w.Name)] = d.(*zos.ZMachine).Network.Interfaces[0].IP.String()
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
	envVars := make([]interface{}, 0)
	for key, value := range vm.EnvVars {
		envVar := map[string]interface{}{
			"key": key, "value": value,
		}
		envVars = append(envVars, envVar)
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
	res["publicip"] = vm.PublicIP
	res["flist"] = vm.Flist
	res["computedip"] = vm.ComputedIP
	res["ip"] = vm.IP
	res["mounts"] = mounts
	res["cpu"] = vm.Cpu
	res["memory"] = vm.Memory
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
	res["password"] = z.Password
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
		vms = append(zdbs, zdb.Dictify())
	}
	d.Set("vms", vms)
	d.Set("zdbs", zdbs)
	d.Set("disks", disks)
	d.Set("node", dep.Node)
	d.Set("network_name", dep.NetworkName)
	d.Set("ip_range", dep.IPRange.String())
}
func resourceDeploymentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := validate(d)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error validating deployment"))
	}
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
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
		wl["name"] = workload.Name
		wl["size"] = data.(*zos.ZDB).Size
		wl["mode"] = data.(*zos.ZDB).Mode
		wl["password"] = data.(*zos.ZDB).Password
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
		envVars := make([]interface{}, 0)
		for key, value := range data.Env {
			envVars = append(envVars, map[string]interface{}{
				"key": key, "value": value,
			})
		}
		machineData, err := workload.WorkloadData()
		if err != nil {
			return nil, err
		}
		wl["cpu"] = data.ComputeCapacity.CPU
		wl["memory"] = uint64(data.ComputeCapacity.Memory) / uint64(gridtypes.Megabyte)
		wl["mounts"] = mounts
		wl["name"] = workload.Name
		wl["flist"] = data.FList
		wl["entrypoint"] = data.Entrypoint
		wl["description"] = workload.Description
		wl["env_vars"] = envVars
		wl["ip"] = machineData.(*zos.ZMachine).Network.Interfaces[0].IP.String()
		return wl, nil
	}

	return nil, errors.New("The wrokload is not of type zos.ZMachineType")
}

func resourceDeploymentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta valufreeIPe to retrieve your client from the provider configure method
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
	cl := apiClient.rmb
	var diags diag.Diagnostics
	sub, err := substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting substrate client"))
	}
	nodeID := uint32(d.Get("node").(int))
	nodeInfo, err := sub.GetNode(nodeID)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting node client"))
	}

	node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)

	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	contractId, err := strconv.ParseUint(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error parsing contract id"))
	}

	deployment, err := node.DeploymentGet(ctx, contractId)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting deployment"))
	}

	disks := make([]map[string]interface{}, 0)
	zdbs := make([]map[string]interface{}, 0)
	vms := make([]map[string]interface{}, 0)
	publicIPs := make(map[string]string)
	for _, workload := range deployment.Workloads {
		if workload.Type == zos.ZMountType {
			flattened, err := flattenDiskData(workload)
			if err != nil {
				return diag.FromErr(errors.Wrap(err, "error flattening disk"))
			}
			disks = append(disks, flattened)

		}
		if workload.Type == zos.ZDBType {
			flattened, err := flattenZDBData(workload)
			if err != nil {
				return diag.FromErr(errors.Wrap(err, "error flattening zdb"))
			}
			zdbs = append(zdbs, flattened)

		} else if workload.Type == zos.ZMachineType {
			flattened, err := flattenVMData(workload)
			if err != nil {
				return diag.FromErr(errors.Wrap(err, "error flattening vm"))
			}
			vms = append(vms, flattened)
		} else if workload.Type == zos.PublicIPType {
			ipData := PubIPData{}
			if err := json.Unmarshal(workload.Result.Data, &ipData); err != nil {
				log.Printf("error unmarshalling json: %s\n", err)
				continue
			}
			publicIPs[string(workload.Name)] = ipData.IP
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
	return diags
}

func validate(d *schema.ResourceData) error {
	ipRangeStr := d.Get("ip_range").(string)
	networkName := d.Get("network_name").(string)
	vms := d.Get("vms").([]interface{})
	_, err := gridtypes.ParseIPNet(ipRangeStr)
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
		return diag.FromErr(errors.New("changing node is not supported, you need to destroy the deployment and reapply it again but you will lost your old data"))
	}
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
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
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
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
