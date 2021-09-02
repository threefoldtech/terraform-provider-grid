package provider

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"reflect"
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
			"version": {
				Description: "Version",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},

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
						"version": {
							Description: "Version",
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
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
						"version": {
							Description: "Version",
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
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
						"version": {
							Description: "Version",
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
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
	Name       string
	Flist      string
	PublicIP   bool
	ComputedIP string
	IP         string
	Cpu        int
	Memory     int
	Entrypoint string
	Mounts     []Mount
	EnvVars    map[string]string
}

type Mount struct {
	DiskName   string
	MountPoint string
}
type DeploymentDeployer struct {
	Node        uint32
	Disks       []Disk
	ZDBs        []ZDB
	VMs         []VM
	IPRange     string
	NetworkName string
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

func resourceDeploymentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := validate(d)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error validating deployment"))
	}

	ipRangeStr := d.Get("ip_range").(string)
	ipRange, err := gridtypes.ParseIPNet(ipRangeStr)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error parsing ip range"))
	}
	usedIPs := make([]string, 0)
	vms := d.Get("vms").([]interface{})
	for _, vm := range vms {
		data := vm.(map[string]interface{})
		usedIPs = append(usedIPs, data["ip"].(string))
	}
	networkName := d.Get("network_name").(string)
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
	identity, err := substrate.IdentityFromPhrase(string(apiClient.mnemonics))
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting identity from phrase"))
	}
	userSK, err := identity.SecureKey()
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting identity's secret key"))
	}

	cl := apiClient.rmb

	var diags diag.Diagnostics
	// twinID := d.Get("twinid").(string)
	nodeID := uint32(d.Get("node").(int))

	disks := d.Get("disks").([]interface{})

	workloads := []gridtypes.Workload{}
	updated_disks := make([]interface{}, 0)
	for _, disk := range disks {
		data := disk.(map[string]interface{})
		data["version"] = Version
		workload := gridtypes.Workload{
			Name:        gridtypes.Name(data["name"].(string)),
			Version:     Version,
			Type:        zos.ZMountType,
			Description: data["description"].(string),
			Data: gridtypes.MustMarshal(zos.ZMount{
				Size: gridtypes.Unit(data["size"].(int)) * gridtypes.Gigabyte,
			}),
		}
		updated_disks = append(updated_disks, data)
		workloads = append(workloads, workload)
	}
	d.Set("disks", updated_disks)

	zdbs := d.Get("zdbs").([]interface{})
	updated_zdbs := make([]interface{}, 0)
	for _, zdb := range zdbs {
		data := zdb.(map[string]interface{})
		data["version"] = Version
		workload := gridtypes.Workload{
			Name:        gridtypes.Name(data["name"].(string)),
			Type:        zos.ZDBType,
			Description: data["description"].(string),
			Version:     Version,
			Data: gridtypes.MustMarshal(zos.ZDB{
				Size:     gridtypes.Unit(data["size"].(int)),
				Mode:     zos.ZDBMode(data["mode"].(string)),
				Password: data["password"].(string),
			}),
		}
		updated_zdbs = append(updated_zdbs, data)
		workloads = append(workloads, workload)
	}
	d.Set("zdbs", updated_zdbs)
	publicIPCount := 0
	updated_vms := make([]interface{}, 0)
	for _, vm := range vms {
		data := vm.(map[string]interface{})
		data["version"] = Version
		mount_points := data["mounts"].([]interface{})
		mounts := []zos.MachineMount{}
		for _, mount_point := range mount_points {
			point := mount_point.(map[string]interface{})
			mount := zos.MachineMount{Name: gridtypes.Name(point["disk_name"].(string)), Mountpoint: point["mount_point"].(string)}
			mounts = append(mounts, mount)
		}
		if data["publicip"].(bool) {
			publicIPCount += 1
			publicIPName := fmt.Sprintf("%sip", data["name"].(string))
			workloads = append(workloads, constructPublicIPWorkload(publicIPName))
		}
		env_vars := data["env_vars"].([]interface{})
		envVars := make(map[string]string)

		for _, env_var := range env_vars {
			envVar := env_var.(map[string]interface{})
			envVars[envVar["key"].(string)] = envVar["value"].(string)
		}
		freeIP, err := getFreeIP(ipRange, usedIPs)
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "error getting free ip"))
		}
		usedIPs = append(usedIPs, freeIP)
		data["ip"] = freeIP
		w := zos.MachineNetwork{
			Interfaces: []zos.MachineInterface{
				{
					Network: gridtypes.Name(networkName),
					IP:      net.ParseIP(freeIP),
				},
			},
			Planetary: true,
		}
		if data["publicip"].(bool) {
			w.PublicIP = gridtypes.Name(fmt.Sprintf("%sip", data["name"].(string)))
		}
		workload := gridtypes.Workload{
			Version: Version,
			Name:    gridtypes.Name(data["name"].(string)),
			Type:    zos.ZMachineType,
			Data: gridtypes.MustMarshal(zos.ZMachine{
				FList:   data["flist"].(string),
				Network: w,
				ComputeCapacity: zos.MachineCapacity{
					CPU:    uint8(data["cpu"].(int)),
					Memory: gridtypes.Unit(uint(data["memory"].(int))) * gridtypes.Megabyte,
				},
				Entrypoint: data["entrypoint"].(string),
				Mounts:     mounts,
				Env:        envVars,
			}),
		}
		updated_vms = append(updated_vms, data)
		workloads = append(workloads, workload)

	}

	d.Set("vms", updated_vms)

	dl := gridtypes.Deployment{
		Version: Version,
		TwinID:  uint32(apiClient.twin_id), //LocalTwin,
		// this contract id must match the one on substrate
		Workloads: workloads,
		SignatureRequirement: gridtypes.SignatureRequirement{
			WeightRequired: 1,
			Requests: []gridtypes.SignatureRequest{
				{
					TwinID: apiClient.twin_id,
					Weight: 1,
				},
			},
		},
	}
	if err := dl.Valid(); err != nil {
		return diag.FromErr(errors.New("invalid: " + err.Error()))
	}
	//return
	if err := dl.Sign(apiClient.twin_id, userSK); err != nil {
		return diag.FromErr(errors.Wrap(err, "error signing deployment"))
	}

	hash, err := dl.ChallengeHash()
	log.Printf("[DEBUG] HASH: %#v", hash)

	if err != nil {
		return diag.FromErr(errors.New("failed to create hash"))
	}

	hashHex := hex.EncodeToString(hash)
	fmt.Printf("hash: %s\n", hashHex)
	// create contract
	sub, err := substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting substrate client"))
	}
	nodeInfo, err := sub.GetNode(nodeID)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting node info"))
	}

	node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)

	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	log.Printf("[DEBUG] NodeId: %#v", nodeID)
	log.Printf("[DEBUG] HASH: %#v", hashHex)
	contractID, err := sub.CreateContract(&identity, nodeID, nil, hashHex, uint32(publicIPCount))
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error creating contract"))
	}
	dl.ContractID = contractID // from substrate

	err = node.DeploymentDeploy(ctx, dl)
	if err != nil {
		cancelDeployment(ctx, node, sub, identity, contractID)
		return diag.FromErr(errors.Wrap(err, "error deploying deployment"))
	}
	err = waitDeployment(ctx, node, dl.ContractID, Version)
	if err != nil {
		cancelDeployment(ctx, node, sub, identity, contractID)
		return diag.FromErr(errors.Wrap(err, "error waiting for deployment"))
	}
	got, err := node.DeploymentGet(ctx, dl.ContractID)
	if err != nil {
		cancelDeployment(ctx, node, sub, identity, contractID)
		return diag.FromErr(errors.Wrap(err, "error getting deployment"))
	}
	pubIP := make(map[string]string)
	for _, wl := range got.Workloads {
		if wl.Type != zos.PublicIPType {
			continue
		}
		d := PubIPData{}
		if err := json.Unmarshal(wl.Result.Data, &d); err != nil {
			return diag.FromErr(errors.Wrap(err, "error unmarshalling"))
		}
		pubIP[string(wl.Name)] = d.IP

	}

	updated_vms2 := make([]interface{}, 0)

	for _, vm := range updated_vms {
		data := vm.(map[string]interface{})
		data["version"] = Version
		ip, ok := pubIP[fmt.Sprintf("%sip", data["name"].(string))]
		if ok {
			data["computedip"] = ip
		}
		updated_vms2 = append(updated_vms2, data)
	}
	d.Set("vms", updated_vms2)
	enc := json.NewEncoder(log.Writer())
	enc.SetIndent("", "  ")
	enc.Encode(got)
	d.SetId(strconv.FormatUint(contractID, 10))
	// resourceDiskRead(ctx, d, meta)

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
		wl["version"] = workload.Version
		return wl, nil
	}

	return nil, errors.New("The wrokload is not of type zos.ZMountType")
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
		for diskName, mountPoint := range data.Mounts {
			mount := map[string]interface{}{
				"disk_name": diskName, "mount_point": mountPoint,
			}
			mounts = append(mounts, mount)
		}
		wl["cpu"] = data.ComputeCapacity.CPU
		wl["memory"] = data.ComputeCapacity.Memory
		wl["mounts"] = mounts
		wl["name"] = workload.Name
		wl["flist"] = data.FList
		wl["version"] = workload.Version
		wl["description"] = workload.Description
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
	vms := make([]map[string]interface{}, 0)
	for _, workload := range deployment.Workloads {
		if workload.Type == zos.ZMountType {
			flattened, err := flattenDiskData(workload)
			if err != nil {
				return diag.FromErr(errors.Wrap(err, "error flattening disk"))
			}
			disks = append(disks, flattened)

		}
		if workload.Type == zos.ZMachineType {
			flattened, err := flattenVMData(workload)
			if err != nil {
				return diag.FromErr(errors.Wrap(err, "error flattening vm"))
			}
			vms = append(vms, flattened)
		}
	}
	d.Set("vms", vms)
	d.Set("disks", disks)
	d.Set("version", deployment.Version)
	return diags
}

func diskHasChanged(disk map[string]interface{}, oldDisks []interface{}) (bool, map[string]interface{}) {
	for _, d := range oldDisks {
		diskData := d.(map[string]interface{})
		if diskData["name"] == disk["name"] {
			if diskData["description"] != disk["description"] || diskData["size"] != disk["size"] {
				return true, diskData
			} else {
				return false, diskData
			}

		}

	}
	return false, nil
}
func zdbHasChanged(zdb map[string]interface{}, oldZdbs []interface{}) (bool, map[string]interface{}) {
	for _, d := range oldZdbs {
		zdbData := d.(map[string]interface{})
		if zdbData["name"] == zdb["name"] {
			if zdbData["password"] != zdb["password"] || zdbData["size"] != zdb["size"] || zdbData["description"] != zdb["description"] || zdbData["mode"] != zdb["mode"] {
				return true, zdbData
			} else {
				return false, zdbData
			}

		}

	}
	return false, nil
}

func vmHasChanged(vm map[string]interface{}, oldVms []interface{}) (bool, map[string]interface{}) {
	for _, machine := range oldVms {
		vmData := machine.(map[string]interface{})
		if vmData["name"] == vm["name"] && vmData["flist"] == vm["flist"] {
			// if vmData.HasChange("cpu") || vmData.HasChange("memory") || vmData.HasChange("entrypoint") || vmData.HasChange("mounts") || vmData.HasChange("env_vars") {
			if vmData["cpu"] != vm["cpu"] || vmData["memory"] != vm["memory"] || vmData["entrypoint"] != vm["entrypoint"] || reflect.DeepEqual(vmData["mounts"], vm["mounts"]) || reflect.DeepEqual(vmData["env_vars"], vm["env_vars"]) {
				return true, vmData
			} else {
				return false, vmData
			}

		}

	}
	return false, nil

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
	ipRangeStr := d.Get("ip_range").(string)
	networkName := d.Get("network_name").(string)
	ipRange, err := gridtypes.ParseIPNet(ipRangeStr)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error parsing ip range"))
	}
	usedIPs := make([]string, 0)
	vms := d.Get("vms").([]interface{})
	for _, vm := range vms {
		data := vm.(map[string]interface{})
		usedIPs = append(usedIPs, data["ip"].(string))
	}

	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
	identity, err := substrate.IdentityFromPhrase(string(apiClient.mnemonics))
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting identity"))
	}
	userSK, err := identity.SecureKey()
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting user sk"))
	}
	cl := apiClient.rmb

	var diags diag.Diagnostics
	// twinID := d.Get("twinid").(string)
	if d.HasChange("node") {
		return diag.FromErr(errors.New("changing node is not supported, you need to destroy the deployment and reapply it again"))
	}
	deploymentHasChange := false
	disksHasChange := false
	zdbsHasChange := false

	if d.HasChange("newtork_name") {
		deploymentHasChange = true
		disksHasChange = true
	}
	if d.HasChange("ip_range") {
		deploymentHasChange = true
		disksHasChange = true
	}
	if d.HasChange("disks") {
		deploymentHasChange = true
		disksHasChange = true
	}
	if d.HasChange("vms") {
		deploymentHasChange = true
	}

	if d.HasChange("zdbs") {
		deploymentHasChange = true
		zdbsHasChange = true
	}
	oldDisks, _ := d.GetChange("disks")
	oldVms, _ := d.GetChange("vms")
	nodeID := uint32(d.Get("node").(int))

	disks := d.Get("disks").([]interface{})
	updatedDisks := make([]interface{}, 0)

	workloads := []gridtypes.Workload{}
	// workloads = append(workloads, network())
	for _, disk := range disks {
		data := disk.(map[string]interface{})
		version := 0
		if disksHasChange {

			changed, oldDisk := diskHasChanged(data, oldDisks.([]interface{}))
			if changed && oldDisk != nil {
				version = oldDisk["version"].(int) + 1
			} else if !changed && oldDisk != nil {
				version = oldDisk["version"].(int)
			}
		}
		data["version"] = version
		workload := gridtypes.Workload{
			Name:        gridtypes.Name(data["name"].(string)),
			Version:     version,
			Type:        zos.ZMountType,
			Description: data["description"].(string),
			Data: gridtypes.MustMarshal(zos.ZMount{
				Size: gridtypes.Unit(data["size"].(int)) * gridtypes.Gigabyte,
			}),
		}
		workloads = append(workloads, workload)
		updatedDisks = append(updatedDisks, data)
	}
	d.Set("disks", updatedDisks)

	oldZdbs, _ := d.GetChange("zdbs")
	zdbs := d.Get("zdbs").([]interface{})
	updatedZdbs := make([]interface{}, 0)
	for _, zdb := range zdbs {
		data := zdb.(map[string]interface{})
		version := 0
		if zdbsHasChange {

			changed, oldZdb := zdbHasChanged(data, oldZdbs.([]interface{}))
			if changed && oldZdb != nil {
				version = oldZdb["version"].(int) + 1
			} else if !changed && oldZdb != nil {
				version = oldZdb["version"].(int)
			}
		}
		data["version"] = version
		workload := gridtypes.Workload{
			Type:        zos.ZDBType,
			Name:        gridtypes.Name(data["name"].(string)),
			Description: data["description"].(string),
			Version:     Version,
			Data: gridtypes.MustMarshal(zos.ZDB{
				Size:     gridtypes.Unit(data["size"].(int)),
				Mode:     zos.ZDBMode(data["mode"].(string)),
				Password: data["password"].(string),
			}),
		}
		workloads = append(workloads, workload)
		updatedZdbs = append(updatedZdbs, data)
	}
	d.Set("zdbs", updatedZdbs)

	updatedVms := make([]interface{}, 0)
	for _, vm := range vms {
		data := vm.(map[string]interface{})
		version := 0
		ip, err := getFreeIP(ipRange, usedIPs)
		changed, oldVmachine := vmHasChanged(data, oldVms.([]interface{}))
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "error when checking whether vm changed"))
		}
		if changed && oldVmachine != nil {
			version = oldVmachine["version"].(int) + 1
			ip = oldVmachine["ip"].(string)
		} else if !changed && oldVmachine != nil {
			version = oldVmachine["version"].(int)
			ip = oldVmachine["ip"].(string)
		} else {
			usedIPs = append(usedIPs, ip)
		}

		data["version"] = version
		data["computedip"] = ""
		mount_points := data["mounts"].([]interface{})
		mounts := []zos.MachineMount{}
		for _, mount_point := range mount_points {
			point := mount_point.(map[string]interface{})
			mount := zos.MachineMount{Name: gridtypes.Name(point["disk_name"].(string)), Mountpoint: point["mount_point"].(string)}
			mounts = append(mounts, mount)
		}

		if data["publicip"].(bool) {
			publicIPName := fmt.Sprintf("%sip", data["name"].(string))
			workloads = append(workloads, constructPublicIPWorkload(publicIPName))
		}

		env_vars := data["env_vars"].([]interface{})
		envVars := make(map[string]string)

		for _, env_var := range env_vars {
			envVar := env_var.(map[string]interface{})
			envVars[envVar["key"].(string)] = envVar["value"].(string)
		}
		workloadData := zos.ZMachine{
			FList: data["flist"].(string),
			Network: zos.MachineNetwork{
				Interfaces: []zos.MachineInterface{
					{
						Network: gridtypes.Name(networkName),
						IP:      net.ParseIP(ip),
					},
				},
				Planetary: true,
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    uint8(data["cpu"].(int)),
				Memory: gridtypes.Unit(uint(data["memory"].(int))) * gridtypes.Megabyte,
			},
			Entrypoint: data["entrypoint"].(string),
			Mounts:     mounts,
			Env:        envVars,
		}
		if data["publicip"].(bool) {
			workloadData.Network.PublicIP = gridtypes.Name(fmt.Sprintf("%sip", data["name"].(string)))
		}
		workload := gridtypes.Workload{
			Version: version,
			Name:    gridtypes.Name(data["name"].(string)),
			Type:    zos.ZMachineType,
			Data:    gridtypes.MustMarshal(workloadData),
		}
		workloads = append(workloads, workload)
		updatedVms = append(updatedVms, data)
	}
	d.Set("vms", updatedVms)
	dlVersion := d.Get("version").(int)
	if deploymentHasChange {
		dlVersion = dlVersion + 1
	}

	dl := gridtypes.Deployment{
		Version: dlVersion,
		TwinID:  uint32(apiClient.twin_id), //LocalTwin,
		// this contract id must match the one on substrate
		Workloads: workloads,
		SignatureRequirement: gridtypes.SignatureRequirement{
			WeightRequired: 1,
			Requests: []gridtypes.SignatureRequest{
				{
					TwinID: apiClient.twin_id,
					Weight: 1,
				},
			},
		},
	}

	if err := dl.Valid(); err != nil {
		return diag.FromErr(errors.New("invalid: " + err.Error()))
	}
	//return
	if err := dl.Sign(apiClient.twin_id, userSK); err != nil {
		return diag.FromErr(errors.Wrap(err, "error signing deployment"))
	}

	hash, err := dl.ChallengeHash()
	log.Printf("[DEBUG] HASH: %#v", hash)

	if err != nil {
		return diag.FromErr(errors.New("failed to create hash"))
	}

	hashHex := hex.EncodeToString(hash)
	fmt.Printf("hash: %s\n", hashHex)
	// create contract
	sub, err := substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting substrate client"))
	}
	nodeInfo, err := sub.GetNode(nodeID)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting node info"))
	}

	node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)

	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	total, used, err := node.Counters(ctx)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting node counters"))
	}

	fmt.Printf("Total: %+v\nUsed: %+v\n", total, used)
	contractID, err := strconv.ParseUint(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error parsing contract id"))
	}
	contractID, err = sub.UpdateContract(&identity, contractID, nil, hashHex)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error updating contract"))
	}
	dl.ContractID = contractID // from substrate

	err = node.DeploymentUpdate(ctx, dl)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error updating deployment"))
	}

	err = waitDeployment(ctx, node, dl.ContractID, dlVersion)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error waiting for deployment"))
	}

	got, err := node.DeploymentGet(ctx, dl.ContractID)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting deployment"))
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(got)
	d.SetId(strconv.FormatUint(contractID, 10))
	// resourceDiskRead(ctx, d, meta)

	pubIP := make(map[string]string)
	for _, wl := range got.Workloads {
		if wl.Type != zos.PublicIPType {
			continue
		}
		d := PubIPData{}
		if err := json.Unmarshal(wl.Result.Data, &d); err != nil {
			return diag.FromErr(errors.Wrap(err, "error unmarshalling"))
		}
		pubIP[string(wl.Name)] = d.IP

	}

	updatedVms2 := make([]interface{}, 0)

	for _, vm := range updatedVms {
		data := vm.(map[string]interface{})
		data["version"] = Version
		ip, ok := pubIP[fmt.Sprintf("%sip", data["name"].(string))]
		if ok {
			data["computedip"] = ip
		}
		updatedVms2 = append(updatedVms2, data)
	}
	d.Set("vms", updatedVms2)
	return diags
}

func resourceDeploymentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	rmbctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go startRmb(rmbctx, apiClient.substrate_url, int(apiClient.twin_id))
	nodeID := uint32(d.Get("node").(int))
	identity, err := substrate.IdentityFromPhrase(string(apiClient.mnemonics))
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting identity"))
	}

	cl := apiClient.rmb
	sub, err := substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting substrate client"))
	}
	nodeInfo, err := sub.GetNode(nodeID)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting node info"))
	}

	node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)

	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	contractID, err := strconv.ParseUint(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error parsing contract id"))
	}
	err = sub.CancelContract(&identity, contractID)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error cancelling contract"))
	}

	err = node.DeploymentDelete(ctx, contractID)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error deleting deployment"))
	}
	d.SetId("")

	return diags

}
