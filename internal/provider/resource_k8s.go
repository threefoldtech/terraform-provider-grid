package provider

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
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

func resourceKubernetes() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Sample resource in the Terraform provider scaffolding.",

		CreateContext: resourceK8sCreate,
		ReadContext:   resourceK8sRead,
		UpdateContext: resourceK8sUpdate,
		DeleteContext: resourceK8sDelete,

		Schema: map[string]*schema.Schema{
			"node_deployment_id": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeInt},
			},
			"network_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ssh_key": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"token": {
				Description: "The cluster secret token",
				Type:        schema.TypeString,
				Required:    true,
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
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"version": {
							Description: "Version",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						"node": {
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
						"flist": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "https://hub.grid.tf/ahmed_hanafy_1/ahmedhanafy725-k3s-latest.flist",
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
							Required:    true,
						},
						"memory": {
							Description: "Memory size",
							Type:        schema.TypeInt,
							Required:    true,
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
							Computed: true,
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
			"workers": {
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
							Optional: true,
							Default:  "https://hub.grid.tf/ahmed_hanafy_1/ahmedhanafy725-k3s-latest.flist",
						},
						"version": {
							Description: "Version",
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
						},
						"disk_size": {
							Description: "Data disk size",
							Type:        schema.TypeInt,
							Required:    true,
						},
						"node": {
							Description: "Node ID",
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
						"ip": {
							Description: "IP",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						"cpu": {
							Description: "CPU size",
							Type:        schema.TypeInt,
							Required:    true,
						},
						"memory": {
							Description: "Memory size",
							Type:        schema.TypeInt,
							Required:    true,
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
							Computed: true,
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
		},
	}
}

func generateMasterWorkload(data map[string]interface{}, IP string, networkName string, SSHKey string, token string) []gridtypes.Workload {

	workloads := make([]gridtypes.Workload, 0)
	size := data["disk_size"].(int)
	version := data["version"].(int)
	masterName := data["name"].(string)
	publicip := data["publicip"].(bool)
	diskWorkload := gridtypes.Workload{
		Name:        "masterdisk",
		Version:     0,
		Type:        zos.ZMountType,
		Description: "Master disk",
		Data: gridtypes.MustMarshal(zos.ZMount{
			Size: gridtypes.Unit(size) * gridtypes.Gigabyte,
		}),
	}
	workloads = append(workloads, diskWorkload)
	publicIPName := ""
	if publicip {
		publicIPName = fmt.Sprintf("%sip", masterName)
		workloads = append(workloads, constructPublicIPWorkload(publicIPName))
	}
	data["version"] = version
	data["ip"] = IP
	envVars := map[string]string{
		"SSH_KEY":           SSHKey,
		"K3S_TOKEN":         token,
		"K3S_DATA_DIR":      "/mydisk",
		"K3S_FLANNEL_IFACE": "eth0",
		"K3S_NODE_NAME":     masterName,
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
				PublicIP: gridtypes.Name(publicIPName),
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    uint8(data["cpu"].(int)),
				Memory: gridtypes.Unit(uint(data["memory"].(int))) * gridtypes.Megabyte,
			},
			Entrypoint: "/sbin/zinit init",
			Mounts: []zos.MachineMount{
				{Name: gridtypes.Name("masterdisk"), Mountpoint: "/mydisk"},
			},
			Env: envVars,
		}),
	}
	workloads = append(workloads, workload)

	return workloads
}

func generateWorkerWorkload(data map[string]interface{}, IP string, masterIP string, networkName string, SSHKey string, token string) []gridtypes.Workload {
	workloads := make([]gridtypes.Workload, 0)
	size := data["disk_size"].(int)
	version := data["version"].(int)
	workerName := data["name"].(string)
	diskName := gridtypes.Name(fmt.Sprintf("%sdisk", workerName))
	publicip := data["publicip"].(bool)
	diskWorkload := gridtypes.Workload{
		Name:        diskName,
		Version:     0,
		Type:        zos.ZMountType,
		Description: "Worker disk",
		Data: gridtypes.MustMarshal(zos.ZMount{
			Size: gridtypes.Unit(size) * gridtypes.Gigabyte,
		}),
	}

	workloads = append(workloads, diskWorkload)
	publicIPName := ""
	if publicip {
		publicIPName = fmt.Sprintf("%sip", workerName)
		workloads = append(workloads, constructPublicIPWorkload(publicIPName))
	}
	data["version"] = version
	data["ip"] = IP
	envVars := map[string]string{
		"SSH_KEY":           SSHKey,
		"K3S_TOKEN":         token,
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
				PublicIP: gridtypes.Name(publicIPName),
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    uint8(data["cpu"].(int)),
				Memory: gridtypes.Unit(uint(data["memory"].(int))) * gridtypes.Megabyte,
			},
			Entrypoint: "/sbin/zinit init",
			Mounts: []zos.MachineMount{
				{Name: diskName, Mountpoint: "/mydisk"},
			},
			Env: envVars,
		}),
	}
	workloads = append(workloads, workload)
	return workloads
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
	apiClient := meta.(*apiClient)
	identity, err := substrate.IdentityFromPhrase(string(apiClient.mnemonics))
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting deployment"))
	}
	userSK, err := identity.SecureKey()
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting user sk"))
	}

	cl := apiClient.client
	sub, err := substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting substrate client"))
	}

	var diags diag.Diagnostics
	// twinID := d.Get("twinid").(string)
	// nodeID := uint32(d.Get("node").(int))

	workloadsNodesMap := make(map[uint32][]gridtypes.Workload)

	nodesIPRangeIfs := d.Get("nodes_ip_range").(map[string]interface{})
	nodesIPRange := make(map[uint32]gridtypes.IPNet)
	for k, v := range nodesIPRangeIfs {
		nodeID, err := strconv.Atoi(k)
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "couldn't convert node id from string to int"))
		}
		nodesIPRange[uint32(nodeID)], err = gridtypes.ParseIPNet(v.(string))
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "couldn't parse ip range"))
		}
	}
	usedIPs := make(map[uint32][]string)
	networkName := d.Get("network_name").(string)
	token := d.Get("token").(string)
	SSHKey := d.Get("ssh_key").(string)

	masterList := d.Get("master").([]interface{})
	master := masterList[0].(map[string]interface{})
	master["version"] = 0
	masterNodeID := uint32(master["node"].(int))
	masterIP, err := getK8sFreeIP(nodesIPRange[masterNodeID], usedIPs[masterNodeID])
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't find a free ip"))
	}
	usedIPs[masterNodeID] = append(usedIPs[masterNodeID], masterIP)

	masterWorkloads := generateMasterWorkload(master, masterIP, networkName, SSHKey, token)
	workloadsNodesMap[masterNodeID] = append(workloadsNodesMap[masterNodeID], masterWorkloads...)
	workers := d.Get("workers").([]interface{})
	updatedWorkers := make([]interface{}, 0)
	for _, vm := range workers {
		data := vm.(map[string]interface{})
		nodeID := uint32(data["node"].(int))
		data["version"] = 0
		freeIP, err := getK8sFreeIP(nodesIPRange[nodeID], usedIPs[nodeID])
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "couldn't get worker free ip"))
		}
		usedIPs[nodeID] = append(usedIPs[nodeID], freeIP)
		workerWorkloads := generateWorkerWorkload(data, freeIP, masterIP, networkName, SSHKey, token)
		updatedWorkers = append(updatedWorkers, data)
		workloadsNodesMap[nodeID] = append(workloadsNodesMap[nodeID], workerWorkloads...)

	}
	nodeDeploymentID := make(map[string]interface{})

	revokeDeployments := false
	defer func() {
		log.Printf("executed at all?\n")
		if !revokeDeployments {
			log.Printf("all went well\n")
			return
		}
		log.Printf("delete all\n")
		for nodeID, deploymentID := range nodeDeploymentID {
			nodeID, err := strconv.Atoi(nodeID)
			if err != nil {
				log.Printf("couldn't convert node if to int %d\n", nodeID)
				continue
			}
			nodeClient, err := getNodClient(uint32(nodeID))
			if err != nil {
				log.Printf("couldn't get node client to delete non-successful deployments\n")
				continue
			}
			log.Printf("deleting deployment %d", deploymentID)
			err = cancelDeployment(ctx, nodeClient, sub, identity, deploymentID.(uint64))

			if err != nil {
				log.Printf("couldn't cancel deployment %d because of %s\n", deploymentID, err)
			}
		}
	}()
	pubIP := make(map[string]string)
	for nodeID, workloads := range workloadsNodesMap {

		publicIPCount := 0
		for _, wl := range workloads {
			if wl.Type == zos.PublicIPType {
				publicIPCount += 1
			}
		}
		dl := gridtypes.Deployment{
			Version:   Version,
			TwinID:    uint32(apiClient.twin_id), //LocalTwin,
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
			revokeDeployments = true
			return diag.FromErr(errors.New("invalid: " + err.Error()))
		}
		//return
		if err := dl.Sign(apiClient.twin_id, userSK); err != nil {
			revokeDeployments = true
			return diag.FromErr(errors.Wrap(err, "error signing deployment"))
		}

		hash, err := dl.ChallengeHash()
		log.Printf("[DEBUG] HASH: %#v", hash)

		if err != nil {
			revokeDeployments = true
			return diag.FromErr(errors.New("failed to create hash"))
		}

		hashHex := hex.EncodeToString(hash)
		fmt.Printf("hash: %s\n", hashHex)
		// create contract
		nodeInfo, err := sub.GetNode(nodeID)
		if err != nil {
			revokeDeployments = true
			return diag.FromErr(errors.Wrap(err, "error getting node info"))
		}

		node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		log.Printf("[DEBUG] NodeId: %#v", nodeID)
		log.Printf("[DEBUG] HASH: %#v", hashHex)
		contractID, err := sub.CreateContract(&identity, nodeID, nil, hashHex, uint32(publicIPCount))
		if err != nil {
			revokeDeployments = true
			return diag.FromErr(errors.Wrap(err, "failed to create contract"))
		}
		dl.ContractID = contractID // from substrate
		nodeDeploymentID[fmt.Sprintf("%d", nodeID)] = contractID

		err = node.DeploymentDeploy(ctx, dl)
		if err != nil {
			revokeDeployments = true
			return diag.FromErr(errors.Wrap(err, "failed to deploy deployment"))
		}
		err = waitDeployment(ctx, node, dl.ContractID, Version)
		if err != nil {
			revokeDeployments = true
			return diag.FromErr(errors.Wrap(err, "error waiting for deployment"))
		}
		got, err := node.DeploymentGet(ctx, dl.ContractID)
		if err != nil {
			revokeDeployments = true
			return diag.FromErr(errors.Wrap(err, "error getting deployment"))
		}
		enc := json.NewEncoder(log.Writer())
		enc.SetIndent("", "  ")
		enc.Encode(got)
		for _, wl := range got.Workloads {
			if wl.Type != zos.PublicIPType {
				continue
			}
			d := PubIPData{}
			if err := json.Unmarshal(wl.Result.Data, &d); err != nil {
				return diag.FromErr(errors.Wrap(err, "error unmarshalling json"))
			}
			pubIP[string(wl.Name)] = d.IP

		}

		// resourceDiskRead(ctx, d, meta)
	}
	if master["publicip"].(bool) {
		ipName := fmt.Sprintf("%sip", master["name"].(string))
		master["computedip"] = pubIP[ipName]
	}
	for idx := range workers {
		if !workers[idx].(map[string]interface{})["publicip"].(bool) {
			continue
		}
		ipName := fmt.Sprintf("%sip", workers[idx].(map[string]interface{})["name"].(string))
		workers[idx].(map[string]interface{})["computedip"] = pubIP[ipName]
	}
	d.SetId(uuid.New().String())
	d.Set("workers", updatedWorkers)
	d.Set("master", master)
	d.Set("node_deployment_id", nodeDeploymentID)
	return diags
}

func resourceK8sUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(*apiClient)
	identity, err := substrate.IdentityFromPhrase(string(apiClient.mnemonics))
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting identity from phrase"))
	}
	userSK, err := identity.SecureKey()
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting user sk"))
	}

	cl := apiClient.client
	sub, err := substrate.NewSubstrate(Substrate)

	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting substrate client"))
	}

	var diags diag.Diagnostics
	// twinID := d.Get("twinid").(string)
	// nodeID := uint32(d.Get("node").(int))

	workloadsNodesMap := make(map[uint32][]gridtypes.Workload)

	nodesIPRangeIfs := d.Get("nodes_ip_range").(map[string]interface{})
	nodesIPRange := make(map[uint32]gridtypes.IPNet)
	for k, v := range nodesIPRangeIfs {
		nodeID, err := strconv.Atoi(k)
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "couldn't convert node id from string to int"))
		}
		nodesIPRange[uint32(nodeID)], err = gridtypes.ParseIPNet(v.(string))
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "couldn't parse ip range"))
		}
	}
	usedIPs := make(map[uint32][]string)
	networkName := d.Get("network_name").(string)
	token := d.Get("token").(string)
	SSHKey := d.Get("ssh_key").(string)
	nodeDeploymentID := d.Get("node_deployment_id").(map[string]interface{})
	oldWorkloadHashes := make(map[string]string)
	oldWorkloadVersion := make(map[string]int)
	oldDeployments := make(map[int]gridtypes.Deployment)
	for nodeID, deploymentID := range nodeDeploymentID {
		nodeID, err := strconv.Atoi(nodeID)
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "error parsing node id"))
		}
		nodeClient, err := getNodClient(uint32(nodeID))
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "error getting node client"))
		}
		dl, err := nodeClient.DeploymentGet(ctx, uint64(deploymentID.(int)))
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "error getting deployment"))
		}
		oldDeployments[nodeID] = dl
		for _, w := range dl.Workloads {
			hash := md5.New()
			if err := w.Challenge(hash); err != nil {
				return diag.FromErr(errors.Wrap(err, "couldn't create challenge"))
			}
			wKey := fmt.Sprintf("%d-%s", nodeID, w.Name)
			oldWorkloadHashes[wKey] = string(hash.Sum(nil))
			oldWorkloadVersion[wKey] = w.Version
		}
	}
	masterList := d.Get("master").([]interface{})
	master := masterList[0].(map[string]interface{})

	// oldMaster := d.GetChange("master").([]interface{})[0].(map[string]interface{})
	// masterChanged := hasMasterChanged(master, oldMaster)

	masterNodeID := uint32(master["node"].(int))
	masterIP, err := getK8sFreeIP(nodesIPRange[masterNodeID], usedIPs[masterNodeID])
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't find a free ip"))
	}
	usedIPs[masterNodeID] = append(usedIPs[masterNodeID], masterIP)

	masterWorkloads := generateMasterWorkload(master, masterIP, networkName, SSHKey, token)
	workloadsNodesMap[masterNodeID] = append(workloadsNodesMap[masterNodeID], masterWorkloads...)
	workers := d.Get("workers").([]interface{})
	updatedWorkers := make([]interface{}, 0)
	for _, vm := range workers {
		data := vm.(map[string]interface{})
		nodeID := uint32(data["node"].(int))
		data["version"] = 0
		freeIP, err := getK8sFreeIP(nodesIPRange[nodeID], usedIPs[nodeID])
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "couldn't get worker free ip"))
		}
		usedIPs[nodeID] = append(usedIPs[nodeID], freeIP)
		workerWorkloads := generateWorkerWorkload(data, freeIP, masterIP, networkName, SSHKey, token)
		updatedWorkers = append(updatedWorkers, data)
		workloadsNodesMap[nodeID] = append(workloadsNodesMap[nodeID], workerWorkloads...)

	}
	nodeDeploymentID = make(map[string]interface{})
	pubIP := make(map[string]string)
	for nodeID, workloads := range workloadsNodesMap {
		createDeployment := true
		oldDeployment, ok := oldDeployments[int(nodeID)]
		if ok {
			createDeployment = false
		}
		version := 0
		if !createDeployment {
			version = oldDeployment.Version + 1
		}
		for idx := range workloads {
			if createDeployment {
				workloads[idx].Version = 0
				continue
			}
			name := workloads[idx].Name
			wKey := fmt.Sprintf("%d-%s", nodeID, name)
			oldHash, exists := oldWorkloadHashes[wKey]
			newHashObj := md5.New()
			if err := workloads[idx].Challenge(newHashObj); err != nil {
				return diag.FromErr(errors.Wrap(err, "couldn't get new workload hash"))
			}
			newHash := string(newHashObj.Sum(nil))
			if !exists || oldHash != newHash {
				workloads[idx].Version = version
			} else {
				workloads[idx].Version = oldWorkloadVersion[wKey]
			}
		}
		log.Printf("Creating? %t, id? %d, version: %d\n", createDeployment, oldDeployment.ContractID, version)
		publicIPCount := 0
		for _, wl := range workloads {
			if wl.Type == zos.PublicIPType {
				publicIPCount += 1
			}
		}
		dl := gridtypes.Deployment{
			Version: version,
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
		log.Printf("prepared deployment\n")
		enc := json.NewEncoder(log.Writer())
		enc.SetIndent("", "  ")
		enc.Encode(dl)
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

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		log.Printf("[DEBUG] NodeId: %#v", nodeID)
		log.Printf("[DEBUG] HASH: %#v", hashHex)
		contractID, err := uint64(0), error(nil)
		if createDeployment {
			contractID, err = sub.CreateContract(&identity, nodeID, nil, hashHex, uint32(publicIPCount))
			if err != nil {
				return diag.FromErr(errors.Wrap(err, "error creating contract"))
			}
		} else {
			contractID, err = sub.UpdateContract(&identity, oldDeployment.ContractID, nil, hashHex)
			if err != nil {
				return diag.FromErr(errors.Wrap(err, "failed to update contract"))
			}
		}
		dl.ContractID = contractID // from substrate
		if createDeployment {
			err = node.DeploymentDeploy(ctx, dl)
			if err != nil {
				return diag.FromErr(errors.Wrap(err, "failed to create deployment"))
			}
		} else {
			err = node.DeploymentUpdate(ctx, dl)
			if err != nil {
				return diag.FromErr(errors.Wrap(err, "failed to update deployment"))
			}

		}
		err = waitDeployment(ctx, node, dl.ContractID, version)
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "error waiting deployment"))
		}
		got, err := node.DeploymentGet(ctx, dl.ContractID)
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "error getting deployment"))
		}
		nodeDeploymentID[fmt.Sprintf("%d", nodeID)] = contractID
		enc = json.NewEncoder(log.Writer())
		enc.SetIndent("", "  ")
		enc.Encode(got)
		// resourceDiskRead(ctx, d, meta)

		for _, wl := range got.Workloads {
			if wl.Type != zos.PublicIPType {
				continue
			}
			d := PubIPData{}
			if err := json.Unmarshal(wl.Result.Data, &d); err != nil {
				return diag.FromErr(errors.Wrap(err, "error unmarshalling pubip data"))
			}
			pubIP[string(wl.Name)] = d.IP

		}

	}
	for nodeID, deployment := range oldDeployments {
		if _, ok := workloadsNodesMap[uint32(nodeID)]; ok {
			continue
		}
		nodeClient, err := getNodClient(uint32(nodeID))
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "error getting node client"))
		}
		cancelDeployment(ctx, nodeClient, sub, identity, deployment.ContractID)
	}
	if master["publicip"].(bool) {
		ipName := fmt.Sprintf("%sip", master["name"].(string))
		master["computedip"] = pubIP[ipName]
	}
	for idx := range updatedWorkers {
		if !updatedWorkers[idx].(map[string]interface{})["publicip"].(bool) {
			continue
		}
		ipName := fmt.Sprintf("%sip", updatedWorkers[idx].(map[string]interface{})["name"].(string))
		updatedWorkers[idx].(map[string]interface{})["computedip"] = pubIP[ipName]
	}
	d.Set("workers", updatedWorkers)
	d.Set("master", master)
	d.Set("node_deployment_id", nodeDeploymentID)
	return diags
}

func resourceK8sRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta valufreeIPe to retrieve your client from the provider configure method
	apiClient := meta.(*apiClient)
	cl := apiClient.client

	nodeDeplomentID := d.Get("node_deployment_id").(map[string]interface{})
	master := d.Get("master").([]interface{})[0].(map[string]interface{})
	workers := d.Get("workers").([]interface{})
	var diags diag.Diagnostics
	sub, err := substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting substrate client"))
	}

	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	masterName := master["name"].(string)
	workloadIdx := make(map[string]int)
	for idx, worker := range workers {
		name := worker.(map[string]interface{})["name"].(string)
		workloadIdx[name] = idx
	}

	for nodeID, deploymentID := range nodeDeplomentID {
		nodeID, err := strconv.Atoi(nodeID)

		if err != nil {
			return diag.FromErr(errors.Wrap(err, "error parsing node id"))
		}

		nodeInfo, err := sub.GetNode(uint32(nodeID))
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "error getting node info"))
		}

		node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)
		deployment, err := node.DeploymentGet(ctx, uint64(deploymentID.(int)))
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "error getting deployment"))
		}

		for _, wl := range deployment.Workloads {
			if wl.Type != zos.ZMachineType {
				continue
			}
			data, err := wl.WorkloadData()
			if err != nil {
				return diag.FromErr(errors.Wrap(err, "error getting workload data"))
			}
			machine := data.(*zos.ZMachine)
			if string(wl.Name) == masterName {
				// TODO: disk size
				master["cpu"] = machine.ComputeCapacity.CPU
				master["memory"] = machine.ComputeCapacity.Memory / 1024 / 1024
				master["flist"] = machine.FList
				master["ip"] = machine.Network.Interfaces[0].IP.String() // make sure this doesn't fail when public ip is deployed
				master["node"] = nodeID
				master["publicip"] = machine.Network.PublicIP != ""
				master["version"] = wl.Version
			}
			idx, ok := workloadIdx[string(wl.Name)]
			if !ok {
				// TODO: read the workload info and add it to the worker
				continue
			}

			worker := workers[idx].(map[string]interface{})
			worker["cpu"] = machine.ComputeCapacity.CPU
			worker["memory"] = machine.ComputeCapacity.Memory / 1024 / 1024
			worker["flist"] = machine.FList
			worker["ip"] = machine.Network.Interfaces[0].IP.String() // make sure this doesn't fail when public ip is deployed
			worker["node"] = nodeID
			worker["publicip"] = machine.Network.PublicIP != ""
			worker["version"] = wl.Version
			workers[idx] = worker
		}
	}

	d.Set("workers", workers)
	d.Set("master", []interface{}{master})
	return diags
}

func cancelDeployment(ctx context.Context, nc *client.NodeClient, sc *substrate.Substrate, identity substrate.Identity, id uint64) error {
	err := sc.CancelContract(&identity, id)
	if err != nil {
		return errors.Wrap(err, "error cancelling contract")
	}

	if err := nc.DeploymentDelete(ctx, id); err != nil {
		return errors.Wrap(err, "error deleting deployment")
	}
	return nil
}

func resourceK8sDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	nodeDeplomentID := d.Get("node_deployment_id").(map[string]interface{})
	identity, err := substrate.IdentityFromPhrase(string(apiClient.mnemonics))
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting identity"))
	}

	cl := apiClient.client

	sub, err := substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "error getting substrate client"))
	}

	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	for nodeID, deploymentID := range nodeDeplomentID {
		nodeID, err := strconv.Atoi(nodeID)

		if err != nil {
			return diag.FromErr(errors.Wrap(err, "error parsing node id"))
		}
		nodeInfo, err := sub.GetNode(uint32(nodeID))
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "error getting node info"))
		}

		nodeClient := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)
		err = cancelDeployment(ctx, nodeClient, sub, identity, uint64(deploymentID.(int)))
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "error cancelling deployment"))
		}
	}
	d.Set("node_deployment_id", nil)
	d.SetId("")

	return diags

}
