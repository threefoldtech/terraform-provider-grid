package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDeployments() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for computing deployments requirements.",

		ReadContext:   dataSourceDeploymentsRead,
		UpdateContext: dataSourceDeploymentsUpdate,
		CreateContext: dataSourceDeploymentsCreate,
		DeleteContext: dataSourceDeploymentsDelete,

		Schema: map[string]*schema.Schema{
			"nodes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     schema.TypeInt,
			},
			"network": {
				Type:     schema.TypeString,
				Required: true,
			},
			"deployment": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"farm": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Farm id of deployment",
						},
						"node": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Node id to place the deployment on",
						},
						"solution_provider": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     0,
							Description: "Solution provider ID",
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
									"flist_checksum": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "if present, the flist is rejected if it has a different hash. the flist hash can be found by append",
									},
									"publicip": {
										Type:        schema.TypeBool,
										Optional:    true,
										Description: "true to enable public ip reservation",
									},
									"publicip6": {
										Type:        schema.TypeBool,
										Optional:    true,
										Description: "true to enable public ipv6 reservation",
									},
									"computedip": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The reserved public ip",
									},
									"computedip6": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The reserved public ipv6",
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
									"corex": {
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
										Description: "Enable corex",
									},
									"ygg_ip": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Allocated Yggdrasil IP",
									},
									"zlogs": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Zlogs is a utility workload that allows you to stream `zmachine` logs to a remote location.",
										Elem: &schema.Schema{
											Type:        schema.TypeString,
											Description: "Url of the remote machine receiving logs."},
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
				},
			},
		},
	}
}

func dataSourceDeploymentsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	return nil
}

func dataSourceDeploymentsCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func dataSourceDeploymentsUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func dataSourceDeploymentsDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}