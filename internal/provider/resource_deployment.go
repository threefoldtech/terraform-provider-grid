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
	"github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/substrate"
)

const (
	Version = 0
	// Twin      = 14
	// NodeID = 4
	// Seed      = "d161de46d136d96085906b9f3d40d08b3649c80a3e4d77f0b14d3dc6889e9dcb"
	// Substrate = "wss://explorer.devnet.grid.tf/ws"
	// rmb_url   = "tcp://127.0.0.1:6379"
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
			"disks": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
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
					},
				},
			},
			"vms": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
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
							Required: true,
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
							Required: true,
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

// func deploy(deployment []gridtypes.Workload, apiClient apiClient){

// }
func network() gridtypes.Workload {
	wgKey := "GDU+cjKrHNJS9fodzjFDzNFl5su3kJXTZ3ipPgUjOUE="

	return gridtypes.Workload{
		Version:     0,
		Type:        zos.NetworkType,
		Description: "test network",
		Name:        "network",
		Data: gridtypes.MustMarshal(zos.Network{
			NetworkIPRange: gridtypes.MustParseIPNet("10.1.0.0/16"),
			Subnet:         gridtypes.MustParseIPNet("10.1.1.0/24"),
			WGPrivateKey:   wgKey,
			WGListenPort:   3011,
			Peers: []zos.Peer{
				{
					Subnet:      gridtypes.MustParseIPNet("10.1.2.0/24"),
					WGPublicKey: "4KTvZS2KPWYfMr+GbiUUly0ANVg8jBC7xP9Bl79Z8zM=",
					AllowedIPs: []gridtypes.IPNet{
						gridtypes.MustParseIPNet("10.1.2.0/24"),
						gridtypes.MustParseIPNet("100.64.0.0/16"),
					},
				},
			},
		}),
	}
}
func resourceDeploymentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(*apiClient)
	identity, err := substrate.IdentityFromPhrase(string(apiClient.mnemonics))
	if err != nil {
		panic(err)
	}
	userSK, err := identity.SecureKey()
	if err != nil {
		panic(err)
	}
	cl := apiClient.client

	var diags diag.Diagnostics
	// twinID := d.Get("twinid").(string)
	nodeID := uint32(d.Get("node").(int))

	disks := d.Get("disks").([]interface{})
	vms := d.Get("vms").([]interface{})

	workloads := []gridtypes.Workload{}
	workloads = append(workloads, network())
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

	updated_vms := make([]interface{}, 0, 0)
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

		env_vars := data["env_vars"].([]interface{})
		envVars := make(map[string]string)

		for _, env_var := range env_vars {
			envVar := env_var.(map[string]interface{})
			envVars[envVar["key"].(string)] = envVar["value"].(string)
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
							Network: "network",
							IP:      net.ParseIP("10.1.1.3"),
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
		panic("invalid: " + err.Error())
	}
	//return
	if err := dl.Sign(apiClient.twin_id, userSK); err != nil {
		panic(err)
	}

	hash, err := dl.ChallengeHash()
	log.Printf("[DEBUG] HASH: %#v", hash)

	if err != nil {
		panic("failed to create hash")
	}

	hashHex := hex.EncodeToString(hash)
	fmt.Printf("hash: %s\n", hashHex)
	// create contract
	sub, err := substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		panic(err)
	}
	nodeInfo, err := sub.GetNode(nodeID)
	if err != nil {
		panic(err)
	}

	node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("[DEBUG] NodeId: %#v", nodeID)
	log.Printf("[DEBUG] HASH: %#v", hashHex)
	contractID, err := sub.CreateContract(&identity, nodeID, nil, hashHex, 0)
	if err != nil {
		panic(err)
	}
	dl.ContractID = contractID // from substrate

	err = node.DeploymentDeploy(ctx, dl)
	if err != nil {
		panic(err)
	}

	got, err := node.DeploymentGet(ctx, dl.ContractID)
	if err != nil {
		panic(err)
	}
	enc := json.NewEncoder(log.Writer())
	enc.SetIndent("", "  ")
	enc.Encode(got)
	d.SetId(strconv.FormatUint(contractID, 10))
	// resourceDiskRead(ctx, d, meta)

	return diags
}

func flattenDiskData(workload gridtypes.Workload) map[string]interface{} {
	if workload.Type == zos.ZMountType {
		wl := make(map[string]interface{})
		data, err := workload.WorkloadData()
		if err != nil {
			panic(err)
		}
		wl["name"] = workload.Name
		wl["size"] = data.(*zos.ZMount).Size / gridtypes.Gigabyte
		wl["description"] = workload.Description
		wl["version"] = workload.Version
		return wl
	}

	panic("The wrokload is not of type zos.ZMountType")
}
func flattenVMData(workload gridtypes.Workload) map[string]interface{} {
	if workload.Type == zos.ZMachineType {
		wl := make(map[string]interface{})
		workloadData, err := workload.WorkloadData()
		if err != nil {
			panic(err)
		}
		data := workloadData.(*zos.ZMachine)
		// envVarsData := data.Env
		// envVars := make([]map[string]interface{})
		// for _, var := range(envvarsDaenvVarsData){

		// }

		mounts := make([]map[string]interface{}, 0, 0)
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
		return wl
	}

	return make(map[string]interface{})
}

func resourceDeploymentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	apiClient := meta.(*apiClient)
	cl := apiClient.client
	var diags diag.Diagnostics
	sub, err := substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		panic(err)
	}
	nodeID := uint32(d.Get("node").(int))
	nodeInfo, err := sub.GetNode(nodeID)
	if err != nil {
		panic(err)
	}

	node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	contractId, err := strconv.ParseUint(d.Id(), 10, 64)
	deployment, err := node.DeploymentGet(ctx, contractId)
	if err != nil {
		panic(err)
	}

	disks := make([]map[string]interface{}, 0, 0)
	vms := make([]map[string]interface{}, 0, 0)
	for _, workload := range deployment.Workloads {
		if workload.Type == zos.ZMountType {
			disks = append(disks, flattenDiskData(workload))
		}
		if workload.Type == zos.ZMachineType {
			vms = append(vms, flattenVMData(workload))
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
func resourceDeploymentUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(*apiClient)
	identity, err := substrate.IdentityFromPhrase(string(apiClient.mnemonics))
	if err != nil {
		panic(err)
	}
	userSK, err := identity.SecureKey()
	if err != nil {
		panic(err)
	}
	cl := apiClient.client

	var diags diag.Diagnostics
	// twinID := d.Get("twinid").(string)
	if d.HasChange("node") {
		panic("changing node is not supported, you need to destroy the deployment and reapply it again")
	}
	deploymentHasChange := false
	disksHasChange := false
	vmsHasChange := false

	if d.HasChange("disks") {
		deploymentHasChange = true
		disksHasChange = true
	}
	if d.HasChange("vms") {
		deploymentHasChange = true
		vmsHasChange = true
	}

	oldDisks, _ := d.GetChange("disks")
	oldVms, _ := d.GetChange("vms")
	nodeID := uint32(d.Get("node").(int))

	disks := d.Get("disks").([]interface{})
	vms := d.Get("vms").([]interface{})
	updatedDisks := make([]interface{}, 0)

	workloads := []gridtypes.Workload{}
	workloads = append(workloads, network())
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

	updatedVms := make([]interface{}, 0)
	for _, vm := range vms {
		data := vm.(map[string]interface{})
		version := 0
		if vmsHasChange {

			changed, oldVmachine := vmHasChanged(data, oldVms.([]interface{}))
			if changed && oldVmachine != nil {
				version = oldVmachine["version"].(int) + 1
			} else if !changed && oldVmachine != nil {
				version = oldVmachine["version"].(int)
			}
		}
		data["version"] = version
		mount_points := data["mounts"].([]interface{})
		mounts := []zos.MachineMount{}
		for _, mount_point := range mount_points {
			point := mount_point.(map[string]interface{})
			mount := zos.MachineMount{Name: gridtypes.Name(point["disk_name"].(string)), Mountpoint: point["mount_point"].(string)}
			mounts = append(mounts, mount)
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
							Network: "network",
							IP:      net.ParseIP("10.1.1.3"),
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
				Env: map[string]string{
					"SSH_KEY": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTwULSsUubOq3VPWL6cdrDvexDmjfznGydFPyaNcn7gAL9lRxwFbCDPMj7MbhNSpxxHV2+/iJPQOTVJu4oc1N7bPP3gBCnF51rPrhTpGCt5pBbTzeyNweanhedkKDsCO2mIEh/92Od5Hg512dX4j7Zw6ipRWYSaepapfyoRnNSriW/s3DH/uewezVtL5EuypMdfNngV/u2KZYWoeiwhrY/yEUykQVUwDysW/xUJNP5o+KSTAvNSJatr3FbuCFuCjBSvageOLHePTeUwu6qjqe+Xs4piF1ByO/6cOJ8bt5Vcx0bAtI8/MPApplUU/JWevsPNApvnA/ntffI+u8DCwgP",
				},
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
		panic("invalid: " + err.Error())
	}
	//return
	if err := dl.Sign(apiClient.twin_id, userSK); err != nil {
		panic(err)
	}

	hash, err := dl.ChallengeHash()
	log.Printf("[DEBUG] HASH: %#v", hash)

	if err != nil {
		panic("failed to create hash")
	}

	hashHex := hex.EncodeToString(hash)
	fmt.Printf("hash: %s\n", hashHex)
	// create contract
	sub, err := substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		panic(err)
	}
	nodeInfo, err := sub.GetNode(nodeID)
	if err != nil {
		panic(err)
	}

	node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	total, used, err := node.Counters(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Total: %+v\nUsed: %+v\n", total, used)
	contractID, err := strconv.ParseUint(d.Id(), 10, 64)
	if err != nil {
		panic(err)
	}
	contractID, err = sub.UpdateContract(&identity, contractID, nil, hashHex)
	if err != nil {
		panic(err)
	}
	dl.ContractID = contractID // from substrate

	err = node.DeploymentUpdate(ctx, dl)
	if err != nil {
		panic(err)
	}

	got, err := node.DeploymentGet(ctx, dl.ContractID)
	if err != nil {
		panic(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(got)
	d.SetId(strconv.FormatUint(contractID, 10))
	// resourceDiskRead(ctx, d, meta)

	return diags
}

func resourceDeploymentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	apiClient := meta.(*apiClient)
	nodeID := uint32(d.Get("node").(int))
	identity, err := substrate.IdentityFromPhrase(string(apiClient.mnemonics))
	if err != nil {
		panic(err)
	}

	if err != nil {
		panic(err)
	}
	cl := apiClient.client
	sub, err := substrate.NewSubstrate(apiClient.substrate_url)
	if err != nil {
		panic(err)
	}
	nodeInfo, err := sub.GetNode(nodeID)
	if err != nil {
		panic(err)
	}

	node := client.NewNodeClient(uint32(nodeInfo.TwinID), cl)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	contractID, err := strconv.ParseUint(d.Id(), 10, 64)
	if err != nil {
		panic(err)
	}
	err = sub.CancelContract(&identity, contractID)
	if err != nil {
		panic(err)
	}

	err = node.DeploymentDelete(ctx, contractID)
	if err != nil {
		panic(err)
	}
	d.SetId("")

	return diags

}
