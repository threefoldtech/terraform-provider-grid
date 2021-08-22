package provider

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/substrate"
)

func k8sDeployment() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Sample resource in the Terraform provider scaffolding.",

		CreateContext: resourceK8sCreate,
		ReadContext:   resourceK8sRead,
		UpdateContext: resourceK8sUpdate,
		DeleteContext: resourceK8sDelete,

		Schema: map[string]*schema.Schema{
			"node_deployment_id": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeInt},
			},
			"disks": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"size": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"description": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"version": {
							Description: "Version",
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
						},
						"nodeid": {
							Description: "Node ID",
							Type:        schema.TypeInt,
							Required:    true,
						},
					},
				},
			},
			"nodes_ip_range": {
				Type:     schema.TypeMap,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"master": {
				MaxItems: 1,
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"version": {
							Description: "Version",
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Default:     -1,
						},
						"node_id": {
							Description: "Node ID",
							Type:        schema.TypeInt,
							Required:    true,
						},
						"disk_size": {
							Description: "Data disk size",
							Type:        schema.TypeInt,
							Required:    true,
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
						"token": {
							Description: "The master token",
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
						"mounts": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"disk_name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"mount_point": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"env_vars": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"value": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			"workers": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"flist": &schema.Schema{
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
						"mounts": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"disk_name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"mount_point": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"env_vars": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"value": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func generateMasterWorkload(data map[string]interface{}, IP string, networkName string) []gridtypes.Workload {

	size := data["disk_size"].(int)
	version := data["version"].(int) + 1
	diskWorkload := gridtypes.Workload{
		Name:        "masterdisk",
		Version:     0,
		Type:        zos.ZMountType,
		Description: "Master disk",
		Data: gridtypes.MustMarshal(zos.ZMount{
			Size: gridtypes.Unit(size) * gridtypes.Gigabyte,
		}),
	}

	data["version"] = version
	data["ip"] = IP
	envVars := map[string]string{
		"SSH_KEY":           data["ssh_key"].(string),
		"K3S_TOKEN":         data["token"].(string),
		"K3S_DATA_DIR":      "/mydisk",
		"K3S_FLANNEL_IFACE": "eth0",
		"K3S_NODE_NAME":     "master",
		"K3S_URL":           "",
	}
	workload := gridtypes.Workload{
		Version: Version,
		Name:    gridtypes.Name(data["name"].(string)),
		Type:    zos.ZMachineType,
		Data: gridtypes.MustMarshal(zos.ZMachine{
			FList: data["flist"].(string),
			Network: zos.MachineNetwork{
				Interfaces: []zos.MachineInterface{
					{
						Network: gridtypes.Name(networkName),
						IP:      net.ParseIP(IP),
					},
				},
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    uint8(data["cpu"].(int)),
				Memory: gridtypes.Unit(uint(data["memory"].(int))) * gridtypes.Megabyte,
			},
			Entrypoint: data["entrypoint"].(string),
			Mounts: []zos.MachineMount{
				zos.MachineMount{Name: gridtypes.Name("masterdisk"), Mountpoint: "/mydisk"},
			},
			Env: envVars,
		}),
	}

	return []gridtypes.Workload{workload, diskWorkload}
}

func generateWorkerWorkload(data map[string]interface{}, workerName string, IP string, masterIP string, networkName string) []gridtypes.Workload {

	size := data["disk_size"].(int)
	version := data["version"].(int) + 1
	diskName := gridtypes.Name(fmt.Sprintf("%s-disk", workerName))
	diskWorkload := gridtypes.Workload{
		Name:        diskName,
		Version:     0,
		Type:        zos.ZMountType,
		Description: "Worker disk",
		Data: gridtypes.MustMarshal(zos.ZMount{
			Size: gridtypes.Unit(size) * gridtypes.Gigabyte,
		}),
	}

	data["version"] = version
	data["ip"] = IP
	envVars := map[string]string{
		"SSH_KEY":           data["ssh_key"].(string),
		"K3S_TOKEN":         data["token"].(string),
		"K3S_DATA_DIR":      "/mydisk",
		"K3S_FLANNEL_IFACE": "eth0",
		"K3S_NODE_NAME":     workerName,
		"K3S_URL":           fmt.Sprintf("https://%s:6443", masterIP),
	}
	workload := gridtypes.Workload{
		Version: Version,
		Name:    gridtypes.Name(data["name"].(string)),
		Type:    zos.ZMachineType,
		Data: gridtypes.MustMarshal(zos.ZMachine{
			FList: data["flist"].(string),
			Network: zos.MachineNetwork{
				Interfaces: []zos.MachineInterface{
					{
						Network: gridtypes.Name(networkName),
						IP:      net.ParseIP(IP),
					},
				},
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    uint8(data["cpu"].(int)),
				Memory: gridtypes.Unit(uint(data["memory"].(int))) * gridtypes.Megabyte,
			},
			Entrypoint: data["entrypoint"].(string),
			Mounts: []zos.MachineMount{
				zos.MachineMount{Name: diskName, Mountpoint: "/mydisk"},
			},
			Env: envVars,
		}),
	}

	return []gridtypes.Workload{workload, diskWorkload}
}

func getK8sFreeIP(ipRange gridtypes.IPNet, usedIPs []string) (string, error) {
	i := 254
	l := len(ipRange.IP)
	for i >= 2 {
		ip := ipNet(ipRange.IP[l-4], ipRange.IP[l-3], ipRange.IP[l-2], byte(i), 32)
		ipStr := fmt.Sprintf("%d.%d.%d.%d", ip.IP[l-4], ip.IP[l-3], ip.IP[l-2], ip.IP[l-1])
		log.Printf("ip string: %s\n", ipStr)
		if !isInStr(usedIPs, ipStr) {
			return ipStr, nil
		}
		i -= 1
	}
	return "", errors.New("all ips are used")
}

func resourceK8sCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := validate(d)
	if err != nil {
		return diag.FromErr(err)
	}

	apiClient := meta.(*apiClient)
	identity, err := substrate.IdentityFromPhrase(string(apiClient.mnemonics))
	if err != nil {
		return diag.FromErr(err)
	}
	userSK, err := identity.SecureKey()
	if err != nil {
		return diag.FromErr(err)
	}

	cl := apiClient.client

	var diags diag.Diagnostics
	// twinID := d.Get("twinid").(string)
	// nodeID := uint32(d.Get("node").(int))

	workloadsNodesMap := make(map[uint32][]gridtypes.Workload)

	ipRangeStr := d.Get("ip_range").(string)
	ipRange, err := gridtypes.ParseIPNet(ipRangeStr)
	usedIPs := make([]string, 0)
	masterIP, err := getK8sFreeIP(ipRange, usedIPs)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't find a free ip"))
	}
	usedIPs = append(usedIPs, masterIP)
	networkName := d.Get("network_name").(string)

	publicIPCount := 0
	master := d.Get("master").(map[string]interface{})
	masterWorkloads := generateMasterWorkload(master, masterIP, networkName)
	masterNodeID := uint32(master["node_id"].(int))
	workloadsNodesMap[masterNodeID] = append(workloadsNodesMap[masterNodeID], masterWorkloads...)
	workers := d.Get("workers").([]interface{})
	updatedWorkers := make([]interface{}, 0)
	for idx, vm := range workers {
		data := vm.(map[string]interface{})
		nodeID := uint32(data["node"].(int))
		usedIPs = append(usedIPs, data["ip"].(string))
		data["version"] = Version
		freeIP, err := getK8sFreeIP(ipRange, usedIPs)
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "couldn't get worker free ip"))
		}
		usedIPs = append(usedIPs, freeIP)
		workerName := fmt.Sprintf("worker-%d", idx)
		workerWorkloads := generateWorkerWorkload(data, workerName, freeIP, masterIP, networkName)
		updatedWorkers = append(updatedWorkers, data)
		workloadsNodesMap[nodeID] = append(workloadsNodesMap[nodeID], workerWorkloads...)

	}
	nodeDeploymendID := make(map[string]interface{})
	for nodeID, workloads := range workloadsNodesMap {

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
			return diag.FromErr(err)
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
			return diag.FromErr(err)
		}
		nodeInfo, err := sub.GetNode(nodeID)
		if err != nil {
			return diag.FromErr(err)
		}

		node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		log.Printf("[DEBUG] NodeId: %#v", nodeID)
		log.Printf("[DEBUG] HASH: %#v", hashHex)
		contractID, err := sub.CreateContract(&identity, nodeID, nil, hashHex, uint32(publicIPCount))
		if err != nil {
			return diag.FromErr(err)
		}
		dl.ContractID = contractID // from substrate

		err = node.DeploymentDeploy(ctx, dl)
		if err != nil {
			return diag.FromErr(err)
		}
		err = waitDeployment(ctx, node, dl.ContractID)
		if err != nil {
			return diag.FromErr(err)
		}
		got, err := node.DeploymentGet(ctx, dl.ContractID)
		if err != nil {
			return diag.FromErr(err)
		}
		nodeDeploymendID[fmt.Sprintf("%d", nodeID)] = contractID
		enc := json.NewEncoder(log.Writer())
		enc.SetIndent("", "  ")
		enc.Encode(got)
		// resourceDiskRead(ctx, d, meta)
	}
	d.SetId(uuid.New().String())
	d.Set("workers", updatedWorkers)
	d.Set("master", master)
	return diags
}

func resourceK8sUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := validate(d)
	if err != nil {
		return diag.FromErr(err)
	}
	ipRangeStr := d.Get("ip_range").(string)
	networkName := d.Get("network_name").(string)
	ipRange, err := gridtypes.ParseIPNet(ipRangeStr)
	usedIPs := make([]string, 0)
	vms := d.Get("vms").([]interface{})
	for _, vm := range vms {
		data := vm.(map[string]interface{})
		usedIPs = append(usedIPs, data["ip"].(string))
	}

	apiClient := meta.(*apiClient)
	identity, err := substrate.IdentityFromPhrase(string(apiClient.mnemonics))
	if err != nil {
		return diag.FromErr(err)
	}
	userSK, err := identity.SecureKey()
	if err != nil {
		return diag.FromErr(err)
	}
	cl := apiClient.client

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
			return diag.FromErr(err)
		}
		if changed && oldVmachine != nil {
			version = oldVmachine["version"].(int) + 1
			ip = oldVmachine["ip_range"].(string)
		} else if !changed && oldVmachine != nil {
			version = oldVmachine["version"].(int)
			ip = oldVmachine["ip_range"].(string)
		} else {
			usedIPs = append(usedIPs, ip)
		}

		data["version"] = version
		mount_points := data["mounts"].([]interface{})
		mounts := []zos.MachineMount{}
		for _, mount_point := range mount_points {
			point := mount_point.(map[string]interface{})
			mount := zos.MachineMount{Name: gridtypes.Name(point["disk_name"].(string)), Mountpoint: point["mount_point"].(string)}
			mounts = append(mounts, mount)
		}

		env_vars := data["env_vars"].([]interface{})
		envVars := make(map[string]string)

		for _, env_var := range env_vars {
			envVar := env_var.(map[string]interface{})
			envVars[envVar["key"].(string)] = envVar["value"].(string)
		}
		workload := gridtypes.Workload{
			Version: version,
			Name:    gridtypes.Name(data["name"].(string)),
			Type:    zos.ZMachineType,
			Data: gridtypes.MustMarshal(zos.ZMachine{
				FList: data["flist"].(string),
				Network: zos.MachineNetwork{
					Interfaces: []zos.MachineInterface{
						{
							Network: gridtypes.Name(networkName),
							IP:      net.ParseIP(ip),
						},
					},
					Planetary: true,
					PublicIP:  gridtypes.Name(fmt.Sprintf("%sip", data["name"].(string))),
				},
				ComputeCapacity: zos.MachineCapacity{
					CPU:    uint8(data["cpu"].(int)),
					Memory: gridtypes.Unit(uint(data["memory"].(int))) * gridtypes.Megabyte,
				},
				Entrypoint: data["entrypoint"].(string),
				Mounts:     mounts,
				Env:        envVars,
			}),
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
		return diag.FromErr(err)
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
		return diag.FromErr(err)
	}
	nodeInfo, err := sub.GetNode(nodeID)
	if err != nil {
		return diag.FromErr(err)
	}

	node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	total, used, err := node.Counters(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	fmt.Printf("Total: %+v\nUsed: %+v\n", total, used)
	contractID, err := strconv.ParseUint(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	contractID, err = sub.UpdateContract(&identity, contractID, nil, hashHex)
	if err != nil {
		return diag.FromErr(err)
	}
	dl.ContractID = contractID // from substrate

	err = node.DeploymentUpdate(ctx, dl)
	if err != nil {
		return diag.FromErr(err)
	}

	err = waitDeployment(ctx, node, dl.ContractID)
	if err != nil {
		return diag.FromErr(err)
	}

	got, err := node.DeploymentGet(ctx, dl.ContractID)
	if err != nil {
		return diag.FromErr(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(got)
	d.SetId(strconv.FormatUint(contractID, 10))
	// resourceDiskRead(ctx, d, meta)

	return diags
}

func resourceK8sRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta valufreeIPe to retrieve your client from the provider configure method
	apiClient := meta.(*apiClient)
	cl := apiClient.client
	var diags diag.Diagnostics
	sub, err := substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		return diag.FromErr(err)
	}
	nodeID := uint32(d.Get("node").(int))
	nodeInfo, err := sub.GetNode(nodeID)
	if err != nil {
		return diag.FromErr(err)
	}

	node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	contractId, err := strconv.ParseUint(d.Id(), 10, 64)
	deployment, err := node.DeploymentGet(ctx, contractId)
	if err != nil {
		return diag.FromErr(err)
	}

	disks := make([]map[string]interface{}, 0, 0)
	vms := make([]map[string]interface{}, 0, 0)
	for _, workload := range deployment.Workloads {
		if workload.Type == zos.ZMountType {
			flattened, err := flattenDiskData(workload)
			if err != nil {
				return diag.FromErr(err)
			}
			disks = append(disks, flattened)

		}
		if workload.Type == zos.ZMachineType {
			flattened, err := flattenVMData(workload)
			if err != nil {
				return diag.FromErr(err)
			}
			vms = append(vms, flattened)
		}
	}
	d.Set("vms", vms)
	d.Set("disks", disks)
	d.Set("version", deployment.Version)
	return diags
}

func resourceK8sDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	nodeID := uint32(d.Get("node").(int))
	identity, err := substrate.IdentityFromPhrase(string(apiClient.mnemonics))
	if err != nil {
		return diag.FromErr(err)
	}

	if err != nil {
		return diag.FromErr(err)
	}
	cl := apiClient.client
	sub, err := substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		return diag.FromErr(err)
	}
	nodeInfo, err := sub.GetNode(nodeID)
	if err != nil {
		return diag.FromErr(err)
	}

	node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	contractID, err := strconv.ParseUint(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	err = sub.CancelContract(&identity, contractID)
	if err != nil {
		return diag.FromErr(err)
	}

	err = node.DeploymentDelete(ctx, contractID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return diags

}
