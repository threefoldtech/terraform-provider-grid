// Package provider is the terraform provider
package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
)

func resourceDeployment() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description:   "Resource for deploying multiple workloads like vms (ZMachines), ZDBs, disks, Qsfss, and/or zlogs. A user should specify node id for this deployment, the (already) deployed network that this deployment should be a part of, and the desired workloads configurations.",
		CreateContext: resourceDeploymentCreate,
		ReadContext:   resourceDeploymentRead,
		UpdateContext: resourceDeploymentUpdate,
		DeleteContext: resourceDeploymentDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(45 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"node": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Node id to place the deployment on.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "vm",
				Description: "Solution name for created contract to be consistent across threefold tooling.",
			},
			"solution_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "Virtual Machine",
				Description: "Solution type for created contract to be consistent across threefold tooling.",
			},
			"solution_provider": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "Solution provider ID for the deployed solution which allows the creator of the solution to gain a percentage of the rewards.",
			},
			"ip_range": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "IP range of the node for the wireguard network (e.g. 10.1.2.0/24). Has to have a subnet mask of 24.",
			},
			"network_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Network name of the deployed network resource to connect vms.",
			},
			"disks": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of disk workloads configurations.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Disk workload name. This has to be unique within the deployment.",
						},
						"size": {
							Type:             schema.TypeInt,
							Required:         true,
							Description:      "Disk size in GBs.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 10*1024)),
						},
						"description": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "Description of disk workload.",
						},
					},
				},
			},
			"zdbs": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of ZDB workloads configurations. You can read more about 0-db (ZDB) [here](https://github.com/threefoldtech/0-db/).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "ZDB worklod name. This has to be unique within the deployment.",
						},
						"password": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "ZDB password.",
						},
						"public": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Makes it read-only if password is set, writable if no password set.",
						},
						"size": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Size of the ZDB in GBs.",
						},
						"description": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "ZDB workload description.",
						},
						"mode": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Mode of the ZDB, `user` or `seq`. `user` is the default mode where a user can SET their own keys, like any key-value store. All keys are kept in memory. in `seq` mode, keys are sequential and autoincremented.",
						},
						"ips": {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Computed:    true,
							Description: "Computed IPs of the ZDB. Two IPs are returned: a public IPv6, and a YggIP, in this order",
						},
						"namespace": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Namespace of the ZDB.",
						},
						"port": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Port of the ZDB.",
						},
					},
				},
			},
			"vms": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of vm (ZMachine) workloads configurations.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Vm (zmachine) workload name. This has to be unique within the deployment.",
						},
						"flist": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Flist used on this vm, e.g. https://hub.grid.tf/tf-official-apps/base:latest.flist. All flists could be found in `https://hub.grid.tf/`.",
						},
						"flist_checksum": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "if present, the flist is rejected if it has a different hash.",
						},
						"publicip": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Flag to enable public ipv4 reservation.",
						},
						"publicip6": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Flag to enable public ipv6 reservation.",
						},
						"computedip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The reserved public ipv4 if any.",
						},
						"computedip6": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The reserved public ipv6 if any.",
						},
						"ip": {
							Type:             schema.TypeString,
							Optional:         true,
							Computed:         true,
							Description:      "The private wireguard IP of the vm.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.IsIPAddress),
						},
						"cpu": {
							Type:             schema.TypeInt,
							Optional:         true,
							Default:          1,
							Description:      "Number of virtual CPUs.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 32)),
						},
						"description": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "Description of the vm.",
						},
						"memory": {
							Type:             schema.TypeInt,
							Optional:         true,
							Description:      "Memory size in MB.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(256, 256*1024)),
						},
						"rootfs_size": {
							Type:             schema.TypeInt,
							Optional:         true,
							Description:      "Root file system size in MB.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1024, 10*1024*1024)),
						},
						"entrypoint": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Command to execute as the ZMachine init.",
						},
						"mounts": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "List of vm (ZMachine) mounts. Can reference QSFSs and Disks.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"disk_name": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Name of QSFS or Disk to mount.",
									},
									"mount_point": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Directory to mount the disk on inside the ZMachine.",
									},
								},
							},
						},
						"env_vars": {
							Type:        schema.TypeMap,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "Environment variables to pass to the zmachine.",
						},
						"planetary": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Flag to enable Yggdrasil IP allocation.",
						},
						"corex": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Flag to enable corex. More information about corex could be found [here](https://github.com/threefoldtech/corex)",
						},
						"ygg_ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The allocated Yggdrasil IP.",
						},
						"zlogs": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "List of Zlogs workloads configurations (URLs). Zlogs is a utility workload that allows you to stream `ZMachine` logs to a remote location.",
							Elem: &schema.Schema{
								Type:        schema.TypeString,
								Description: "Url of the remote location receiving logs. URLs should use one of `redis, ws, wss` schema. e.g. wss://example_ip.com:9000"},
						},
					},
				},
			},
			"qsfs": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of Qsfs workloads configurations. Qsfs is a quantum storage file system.\nYou can read more about it [here](https://github.com/threefoldtech/quantum-storage).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Qsfs workload name. This has to be unique within the deployment.",
						},
						"description": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "Description of the qsfs workload.",
						},
						"cache": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The size of the fuse mountpoint on the node in MBs (holds qsfs local data before pushing).",
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
							Description: "64 long hex encoded encryption key (e.g. 0000000000000000000000000000000000000000000000000000000000000000).",
						},
						"compression_algorithm": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "snappy",
							Description: "configuration to use for the compression stage. Currently only snappy is supported.",
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
										Description: "configuration for the metadata store to use, currently only ZDB is supported.",
									},
									"prefix": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Data stored on the remote metadata is prefixed with.",
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
										Description: "64 long hex encoded encryption key (e.g. 0000000000000000000000000000000000000000000000000000000000000000).",
									},
									"backends": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "List of ZDB backends configurations.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"address": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Address of backend ZDB (e.g. [300:a582:c60c:df75:f6da:8a92:d5ed:71ad]:9900 or 60.60.60.60:9900).",
												},
												"namespace": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "ZDB namespace.",
												},
												"password": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Namespace password.",
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
										Type:        schema.TypeList,
										Optional:    true,
										Description: "List of ZDB backends configurations.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"address": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Address of backend ZDB (e.g. [300:a582:c60c:df75:f6da:8a92:d5ed:71ad]:9900 or 60.60.60.60:9900).",
												},
												"namespace": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "ZDB namespace.",
												},
												"password": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Namespace password.",
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
							Description: "QSFS exposed metrics endpoint.",
						},
					},
				},
			},
		},
	}
}

func resourceDeploymentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	dl, err := newDeploymentFromSchema(d)
	if err != nil {
		return diag.Errorf("couldn't load deployment data with error: %v", err)
	}

	if err := tfPluginClient.DeploymentDeployer.Deploy(ctx, dl); err != nil {
		return diag.Errorf("couldn't deploy deployment with error: %v", err)
	}

	if err := tfPluginClient.DeploymentDeployer.Sync(ctx, dl); err != nil {
		return diag.Errorf("couldn't sync deployment with error: %v", err)
	}

	if err := syncContractsDeployments(d, dl); err != nil {
		return diag.Errorf("couldn't set deployment data to the resource with error: %v", err)
	}

	return diags
}

func resourceDeploymentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	dl, err := newDeploymentFromSchema(d)
	if err != nil {
		return diag.Errorf("couldn't load deployment data with error: %v", err)
	}

	if err := tfPluginClient.DeploymentDeployer.Sync(ctx, dl); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "failed to read deployment data (terraform refresh might help)",
			Detail:   err.Error(),
		})
		return diags
	}

	if err := syncContractsDeployments(d, dl); err != nil {
		return diag.Errorf("couldn't set deployment data to the resource with error: %v", err)
	}

	return diags
}

func resourceDeploymentUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	if d.HasChange("node") {
		oldContractID, err := strconv.ParseUint(d.Id(), 10, 64)
		if err != nil {
			return diag.Errorf("couldn't parse deployment id %s with error: %v", d.Id(), err)
		}
		err = tfPluginClient.SubstrateConn.CancelContract(tfPluginClient.Identity, oldContractID)
		if err != nil {
			return diag.Errorf("couldn't cancel old node contract with error: %v", err)
		}
		d.SetId("")
	}

	dl, err := newDeploymentFromSchema(d)
	if err != nil {
		return diag.Errorf("couldn't load deployment data with error: %v", err)
	}

	if err := tfPluginClient.DeploymentDeployer.Deploy(ctx, dl); err != nil {
		return diag.Errorf("couldn't update deployment with error: %v", err)
	}

	if err := tfPluginClient.DeploymentDeployer.Sync(ctx, dl); err != nil {
		return diag.Errorf("couldn't sync deployment with error: %v", err)
	}

	if err := syncContractsDeployments(d, dl); err != nil {
		return diag.Errorf("couldn't set deployment data to the resource with error: %v", err)
	}

	return diags
}

func resourceDeploymentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	dl, err := newDeploymentFromSchema(d)
	if err != nil {
		return diag.Errorf("couldn't load deployment data with error: %v", err)
	}

	if err := tfPluginClient.DeploymentDeployer.Cancel(ctx, dl); err != nil {
		return diag.Errorf("couldn't cancel deployment with error: %v", err)
	}

	if err := tfPluginClient.DeploymentDeployer.Sync(ctx, dl); err != nil {
		return diag.Errorf("couldn't sync deployment with error: %v", err)
	}

	if err := syncContractsDeployments(d, dl); err != nil {
		return diag.Errorf("couldn't set deployment data to the resource with error: %v", err)
	}

	return diags
}
