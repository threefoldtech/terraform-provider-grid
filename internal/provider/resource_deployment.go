// Package provider is the terraform provider
package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
)

func resourceDeployment() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description:   "Resource for deploying multiple workloads like vms (Zmachines), zdbs, disks, Qsfss, and/or zlogs.\nA user should specify node id for this deployment, the (already) deployed network that this deployment should be a part of, and the desired workloads configurations.",
		CreateContext: ResourceFunc(resourceDeploymentCreate),
		ReadContext:   ResourceReadFunc(resourceDeploymentRead),
		UpdateContext: ResourceFunc(resourceDeploymentUpdate),
		DeleteContext: ResourceFunc(resourceDeploymentDelete),

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(45 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"node_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Node id to place the deployment on.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "vm",
				Description: "Solution name for created contract, displayed [here](https://play.dev.grid.tf/#/contractslist).",
			},
			"solution_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "Virtual Machine",
				Description: "Solution type for created contract, displayed [here](https://play.dev.grid.tf/#/contractslist).",
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
							Description: "Disk workload name.",
						},
						"size": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Disk size in GBs.",
						},
						"description": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "Description for disk workload.",
						},
					},
				},
			},
			"zdbs": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of Zdb workloads configurations. You can read more about 0-db (Zdb) [here](https://github.com/threefoldtech/0-db/).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Zdb worklod name.",
						},
						"password": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Zdb password.",
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
							Description: "Size of the zdb in GBs.",
						},
						"description": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "Zdb workload description.",
						},
						"mode": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Mode of the zdb, `user` or `seq`. `user` is the default mode where a user can SET their own keys, like any key-value store. All keys are kept in memory. in `seq` mode, keys are sequential and autoincremented.",
						},
						"ips": {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Computed:    true,
							Description: "Computed IPs of the zdb.",
						},
						"namespace": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Namespace of the zdb.",
						},
						"port": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Port of the zdb.",
						},
					},
				},
			},
			"vms": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of vm (Zmachine) workloads configurations.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Vm (zmachine) workload name.",
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
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "The private wireguard IP of the vm.",
						},
						"cpu": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     1,
							Description: "Number of virtual cpus.",
						},
						"description": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "Description for the vm.",
						},
						"memory": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Memory size in MB.",
						},
						"rootfs_size": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Rootfs size in MB.",
						},
						"entrypoint": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Command to execute as the Zmachine init.",
						},
						"mounts": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "List of vm (Zmachine) mounts. Can reference QSFSs and Disks.",
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
										Description: "Directory to mount the disk on inside the Zmachine.",
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
							Description: "Flag to enable Yggdrasil ip allocation.",
						},
						"corex": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Flag to enable corex.",
						},
						"ygg_ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The allocated Yggdrasil IP.",
						},
						"zlogs": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "List of Zlogs workloads configurations. Zlogs is a utility workload that allows you to stream `zmachine` logs to a remote location.",
							Elem: &schema.Schema{
								Type:        schema.TypeString,
								Description: "Url of the remote machine receiving logs."},
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
							Description: "Qsfs workload name.",
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
										Description: "configuration for the metadata store to use, currently only zdb is supported.",
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
										Description: "List of zdb backends configurations.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"address": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Address of backend zdb (e.g. [300:a582:c60c:df75:f6da:8a92:d5ed:71ad]:9900 or 60.60.60.60:9900).",
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
										Description: "List of zdb backends configurations.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"address": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Address of backend zdb (e.g. [300:a582:c60c:df75:f6da:8a92:d5ed:71ad]:9900 or 60.60.60.60:9900).",
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

func resourceDeploymentCreate(ctx context.Context, sub subi.SubstrateExt, d *schema.ResourceData, apiClient *apiClient) (Marshalable, error) {
	deployer, err := getDeploymentDeployer(d, apiClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}

	return &deployer, deployer.Deploy(ctx, sub)
}

func resourceDeploymentRead(ctx context.Context, sub subi.SubstrateExt, d *schema.ResourceData, apiClient *apiClient) (Marshalable, error) {
	deployer, err := getDeploymentDeployer(d, apiClient)
	if err != nil {
		return nil, err
	}
	return &deployer, nil
}

func resourceDeploymentUpdate(ctx context.Context, sub subi.SubstrateExt, d *schema.ResourceData, apiClient *apiClient) (Marshalable, error) {
	if d.HasChange("node") {
		oldContractID, err := strconv.ParseUint(d.Id(), 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't parse deployment id %s", d.Id())
		}
		err = sub.CancelContract(apiClient.identity, oldContractID)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't cancel old node contract")
		}
		d.SetId("")
	}
	deployer, err := getDeploymentDeployer(d, apiClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}

	return &deployer, deployer.Deploy(ctx, sub)
}

func resourceDeploymentDelete(ctx context.Context, sub subi.SubstrateExt, d *schema.ResourceData, apiClient *apiClient) (Marshalable, error) {
	deployer, err := getDeploymentDeployer(d, apiClient)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load deployer data")
	}

	return &deployer, deployer.Cancel(ctx, sub)
}
