package converter

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

type DeploymentDeployer struct {
	Id               string             `name:"id"`
	Node             int16              `name:"node"`
	Disks            []Disk             `name:"disks"`
	ZDBList          []ZDB              `name:"zdbs"`
	VMList           []VM               `name:"vms"`
	QSFSList         []QSFS             `name:"qsfss"`
	IPRange          string             `name:"ip_range"`
	NetworkName      string             `name:"network_name"`
	FQDN             []GatewayFQDNProxy `name:"fqdn"`
	GatewayNames     []GatewayNameProxy `name:"gateway_names"`
	NodeDeploymentID map[uint32]uint64  `name:"node_deployment_id"`
}

// type DeploymentDeployer struct {
// 	Id          string                      `name:"id"`
// 	Node        uint32                      `name:"node"`
// 	Disks       []workloads.Disk            `name:"disks"`
// 	ZDBs        []workloads.ZDB             `name:"zdbs"`
// 	VMs         []workloads.VM              `name:"vms"`
// 	QSFSs       []workloads.QSFS            `name:"qsfss"`
// 	IPRange     string                      `name:"ip_range"`
// 	NetworkName string                      `name:"network_name"`
// 	APIClient   *apiClient                  `name:"api_client"`
// 	ncPool      client.NodeClientCollection `name:"nc_pool"`
// 	deployer    deployer.Deployer           `name:"deployer"`
// }

type VM struct {
	Name          string            `name:"name"`
	Flist         string            `name:"flist"`
	FlistChecksum string            `name:"flist_checksum"`
	Publicip      bool              `name:"publicip"`
	Publicip6     bool              `name:"publicip6"`
	Planetary     bool              `name:"planetary"`
	Corex         bool              `name:"corex"`
	Computedip    string            `name:"computedip"`
	Computedip6   string            `name:"computedip6"`
	YggIP         string            `name:"ygg_ip"`
	IP            string            `name:"ip"`
	Description   string            `name:"description"`
	Cpu           int               `name:"cpu"`
	Memory        int               `name:"memory"`
	RootfsSize    int               `name:"rootfs_size"`
	Entrypoint    string            `name:"entrypoint"`
	Mounts        []Mount           `name:"mounts"`
	Zlogs         []Zlog            `name:"zlogs"`
	EnvVars       map[string]string `name:"env_vars"`

	NetworkName string `name:"network_name"`
}

type Mount struct {
	DiskName   string `name:"disk_name"`
	MountPoint string `name:"mount_point"`
}

type Zlog struct {
	Output string `name:"output"`
}

type Disk struct {
	Name        string `name:"name"`
	Size        int    `name:"size"`
	Description string `name:"description"`
}
type ZDB struct {
	Name        string   `name:"name"`
	Password    string   `name:"password"`
	Public      bool     `name:"public"`
	Size        int      `name:"size"`
	Description string   `name:"description"`
	Mode        string   `name:"mode"`
	Ips         []string `name:"ips"`
	Port        uint32   `name:"port"`
	Namespace   string   `name:"namespace"`
}

type QSFS struct {
	Name                 string   `name:"name"`
	Description          string   `name:"description"`
	Cache                int      `name:"cache"`
	MinimalShards        uint32   `name:"minimal_shards"`
	ExpectedShards       uint32   `name:"expected_shards"`
	RedundantGroups      uint32   `name:"redundant_groups"`
	RedundantNodes       uint32   `name:"redundant_nodes"`
	MaxZDBDataDirSize    uint32   `name:"max_zdb_data_dir_size"`
	EncryptionAlgorithm  string   `name:"encryption_algorithm"`
	EncryptionKey        string   `name:"encryption_key"`
	CompressionAlgorithm string   `name:"compression_algorithm"`
	Metadata             Metadata `name:"metadata"`
	Groups               Groups   `name:"groups"`

	MetricsEndpoint string `name:"metrics_endpoint"`
}
type Metadata struct {
	Type                string   `name:"type"`
	Prefix              string   `name:"prefix"`
	EncryptionAlgorithm string   `name:"encryption_algorithm"`
	EncryptionKey       string   `name:"encryption_key"`
	Backends            Backends `name:"backends"`
}
type Group struct {
	Backends Backends `name:"backends"`
}
type Backend ZdbBackend
type Groups []Group
type Backends []Backend

type ZdbBackend struct {
	Address   string `name:"address"`
	Namespace string `name:"namespace"`
	Password  string `name:"password"`
}

type GatewayNameProxy struct {
	// Name the fully qualified domain name to use (cannot be present with Name)
	Name string `name:"name"`

	// Passthrough whether to pass tls traffic or not
	TLSPassthrough bool `name:"tls_passthrough"`

	// Backends are list of backend ips
	Backends []Backend `name:"backends"`

	// FQDN deployed on the node
	FQDN string `name:"fqdn"`
}

type GatewayFQDNProxy struct {
	// Name the fully qualified domain name to use (cannot be present with Name)
	Name string `name:"name"`

	// Passthrough whether to pass tls traffic or not
	TLSPassthrough bool `name:"tls_passthrough"`

	// Backends are list of backend ips
	Backends []Backend `name:"backends"`

	// FQDN deployed on the node
	FQDN string `name:"fqdn"`
}

func getDeployment() DeploymentDeployer {
	dp := DeploymentDeployer{}
	dp.Id = "1234"
	dp.Node = int16(1)
	dp.Disks = []Disk{{"d1", 5, "desc1"}, {"d2", 6, "desc2"}}
	dp.ZDBList = []ZDB{{
		"zdb1",
		"pass1",
		true,
		5,
		"desc1",
		"mod1",
		[]string{"ip1, ip2"},
		1234,
		"namespace1",
	},
		{
			"zdb2",
			"pass2",
			true,
			5,
			"desc2",
			"mod2",
			[]string{"ip3, ip4"},
			5678,
			"namespace2",
		},
	}

	dp.VMList = []VM{{
		"vm1",
		"flist1",
		"flist_checksum1",
		false,
		false,
		true,
		false,
		"computedip_1",
		"computedip6_1",
		"yggip1",
		"ip1",
		"desc1",
		2,
		5,
		3,
		"entrypoint1",
		[]Mount{{"d1", "mp1"}, {"d2", "mp2"}},
		[]Zlog{{"zlog1"}, {"zlog2"}},
		map[string]string{"1": "var1", "2": "var2"},
		"net1",
	},
		{
			"vm2",
			"flist2",
			"flist_checksum2",
			true,
			true,
			false,
			true,
			"computedip_2",
			"computedip6_2",
			"yggip2",
			"ip2",
			"desc2",
			5,
			7,
			4,
			"entrypoint2",
			[]Mount{{"d5", "mp5"}, {"d6", "mp6"}},
			[]Zlog{{"zlog3"}, {"zlog4"}},
			map[string]string{"3": "var3", "4": "var4"},
			"net2",
		},
	}
	dp.QSFSList = []QSFS{
		{
			"name1",
			"desc1",
			1,
			2,
			3,
			4,
			5,
			6,
			"encalgo",
			"key1",
			"comalgo",
			Metadata{
				"tp1",
				"pre1",
				"encalgo",
				"key1",
				Backends{{"add3", "ns3", "pss3"}, {"add4", "ns4", "pss4"}},
			},
			Groups{
				{
					Backends{{"add3", "ns3", "pss3"}, {"add4", "ns4", "pss4"}},
				},
				{
					Backends{{"add3", "ns3", "pss3"}, {"add4", "ns4", "pss4"}},
				},
			},
			"endpoint1",
		},
		{
			"name2",
			"desc2",
			1,
			2,
			3,
			4,
			5,
			6,
			"encalgo",
			"key1",
			"comalgo",
			Metadata{
				"tp1",
				"pre1",
				"encalgo",
				"key1",
				Backends{{"add3", "ns3", "pss3"}, {"add4", "ns4", "pss4"}},
			},
			Groups{
				{
					Backends{{"add3", "ns3", "pss3"}, {"add4", "ns4", "pss4"}},
				},
				{
					Backends{{"add3", "ns3", "pss3"}, {"add4", "ns4", "pss4"}},
				},
			},
			"endpoint2",
		},
	}
	dp.IPRange = "iprange"
	dp.NetworkName = "net1"
	dp.FQDN = []GatewayFQDNProxy{
		{
			"name1",
			true,
			[]Backend{{"add3", "ns3", "pss3"}, {"add4", "ns4", "pss4"}},
			"fqdn1",
		},
		{
			"name2",
			true,
			[]Backend{{"add3", "ns3", "pss3"}, {"add4", "ns4", "pss4"}},
			"fqdn2",
		},
	}
	dp.GatewayNames = []GatewayNameProxy{
		{
			"name1",
			false,
			[]Backend{{"add3", "ns3", "pss3"}, {"add4", "ns4", "pss4"}},
			"fqdn1",
		},
		{
			"name2",
			false,
			[]Backend{{"add3", "ns3", "pss3"}, {"add4", "ns4", "pss4"}},
			"fqdn2",
		},
	}
	dp.NodeDeploymentID = map[uint32]uint64{1: 2}

	return dp
}

type ResourceData struct {
	data map[string]interface{}
}

func (r *ResourceData) Get(key string) interface{} {
	return r.data[key]
}

func (r *ResourceData) Set(key string, value interface{}) error {
	r.data[key] = value
	return nil
}

func getFilledResourceData() ResourceData {
	rd := ResourceData{}
	rd.data = map[string]interface{}{}
	rd.Set("id", "1234")
	rd.Set("node", 1)
	rd.Set("disks", []interface{}{
		map[string]interface{}{
			"name":        "d1",
			"size":        int(5),
			"description": "desc1",
		}, map[string]interface{}{
			"name":        "d2",
			"size":        int(6),
			"description": "desc2",
		}})
	rd.Set("zdbs", []interface{}{
		map[string]interface{}{
			"name":        "zdb1",
			"password":    "pass1",
			"public":      true,
			"size":        int(5),
			"description": "desc1",
			"mode":        "mod1",
			"ips":         []interface{}{"ip1, ip2"},
			"port":        int(1234),
			"namespace":   "namespace1",
		},
		map[string]interface{}{
			"name":        "zdb2",
			"password":    "pass2",
			"public":      true,
			"size":        int(5),
			"description": "desc2",
			"mode":        "mod2",
			"ips":         []interface{}{"ip3, ip4"},
			"port":        int(5678),
			"namespace":   "namespace2",
		},
	})
	rd.Set("vms", []interface{}{
		map[string]interface{}{
			"name":           "vm1",
			"flist":          "flist1",
			"flist_checksum": "flist_checksum1",
			"publicip":       false,
			"publicip6":      false,
			"planetary":      true,
			"corex":          false,
			"computedip":     "computedip_1",
			"computedip6":    "computedip6_1",
			"ygg_ip":         "yggip1",
			"ip":             "ip1",
			"description":    "desc1",
			"cpu":            int(2),
			"memory":         int(5),
			"rootfs_size":    int(3),
			"entrypoint":     "entrypoint1",
			"mounts": []interface{}{
				map[string]interface{}{
					"disk_name":   "d1",
					"mount_point": "mp1",
				},
				map[string]interface{}{
					"disk_name":   "d2",
					"mount_point": "mp2",
				}},
			"zlogs":        []interface{}{map[string]interface{}{"output": "zlog1"}, map[string]interface{}{"output": "zlog2"}},
			"env_vars":     map[string]interface{}{"1": "var1", "2": "var2"},
			"network_name": "net1",
		},
		map[string]interface{}{
			"name":           "vm2",
			"flist":          "flist2",
			"flist_checksum": "flist_checksum2",
			"publicip":       true,
			"publicip6":      true,
			"planetary":      false,
			"corex":          true,
			"computedip":     "computedip_2",
			"computedip6":    "computedip6_2",
			"ygg_ip":         "yggip2",
			"ip":             "ip2",
			"description":    "desc2",
			"cpu":            int(5),
			"memory":         int(7),
			"rootfs_size":    int(4),
			"entrypoint":     "entrypoint2",
			"mounts": []interface{}{
				map[string]interface{}{
					"disk_name":   "d5",
					"mount_point": "mp5",
				},
				map[string]interface{}{
					"disk_name":   "d6",
					"mount_point": "mp6",
				}},
			"zlogs":        []interface{}{map[string]interface{}{"output": "zlog3"}, map[string]interface{}{"output": "zlog4"}},
			"env_vars":     map[string]interface{}{"3": "var3", "4": "var4"},
			"network_name": "net2",
		},
	})
	rd.Set("qsfss", []interface{}{
		map[string]interface{}{
			"name":                  "name1",
			"description":           "desc1",
			"cache":                 int(1),
			"minimal_shards":        int(2),
			"expected_shards":       int(3),
			"redundant_groups":      int(4),
			"redundant_nodes":       int(5),
			"max_zdb_data_dir_size": int(6),
			"encryption_algorithm":  "encalgo",
			"encryption_key":        "key1",
			"compression_algorithm": "comalgo",
			"metadata": map[string]interface{}{
				"type":                 "tp1",
				"prefix":               "pre1",
				"encryption_algorithm": "encalgo",
				"encryption_key":       "key1",
				"backends": []interface{}{
					map[string]interface{}{
						"address":   "add3",
						"namespace": "ns3",
						"password":  "pss3",
					},
					map[string]interface{}{
						"address":   "add4",
						"namespace": "ns4",
						"password":  "pss4",
					},
				},
			},
			"groups": []interface{}{
				map[string]interface{}{
					"backends": []interface{}{
						map[string]interface{}{
							"address":   "add3",
							"namespace": "ns3",
							"password":  "pss3",
						},
						map[string]interface{}{
							"address":   "add4",
							"namespace": "ns4",
							"password":  "pss4",
						},
					},
				},
				map[string]interface{}{
					"backends": []interface{}{
						map[string]interface{}{
							"address":   "add3",
							"namespace": "ns3",
							"password":  "pss3",
						},
						map[string]interface{}{
							"address":   "add4",
							"namespace": "ns4",
							"password":  "pss4",
						},
					},
				},
			},
			"metrics_endpoint": "endpoint1",
		},
		map[string]interface{}{
			"name":                  "name2",
			"description":           "desc2",
			"cache":                 int(1),
			"minimal_shards":        int(2),
			"expected_shards":       int(3),
			"redundant_groups":      int(4),
			"redundant_nodes":       int(5),
			"max_zdb_data_dir_size": int(6),
			"encryption_algorithm":  "encalgo",
			"encryption_key":        "key1",
			"compression_algorithm": "comalgo",
			"metadata": map[string]interface{}{
				"type":                 "tp1",
				"prefix":               "pre1",
				"encryption_algorithm": "encalgo",
				"encryption_key":       "key1",
				"backends": []interface{}{
					map[string]interface{}{
						"address":   "add3",
						"namespace": "ns3",
						"password":  "pss3",
					},
					map[string]interface{}{
						"address":   "add4",
						"namespace": "ns4",
						"password":  "pss4",
					},
				},
			},
			"groups": []interface{}{
				map[string]interface{}{
					"backends": []interface{}{
						map[string]interface{}{
							"address":   "add3",
							"namespace": "ns3",
							"password":  "pss3",
						},
						map[string]interface{}{
							"address":   "add4",
							"namespace": "ns4",
							"password":  "pss4",
						},
					},
				},
				map[string]interface{}{
					"backends": []interface{}{
						map[string]interface{}{
							"address":   "add3",
							"namespace": "ns3",
							"password":  "pss3",
						},
						map[string]interface{}{
							"address":   "add4",
							"namespace": "ns4",
							"password":  "pss4",
						},
					},
				},
			},
			"metrics_endpoint": "endpoint2",
		},
	})
	rd.Set("ip_range", "iprange")
	rd.Set("network_name", "net1")
	rd.Set("fqdn", []interface{}{
		map[string]interface{}{
			"name":            "name1",
			"tls_passthrough": true,
			"backends": []interface{}{
				map[string]interface{}{
					"address":   "add3",
					"namespace": "ns3",
					"password":  "pss3",
				},
				map[string]interface{}{
					"address":   "add4",
					"namespace": "ns4",
					"password":  "pss4",
				},
			},
			"fqdn": "fqdn1",
		},
		map[string]interface{}{
			"name":            "name2",
			"tls_passthrough": true,
			"backends": []interface{}{
				map[string]interface{}{
					"address":   "add3",
					"namespace": "ns3",
					"password":  "pss3",
				},
				map[string]interface{}{
					"address":   "add4",
					"namespace": "ns4",
					"password":  "pss4",
				},
			},
			"fqdn": "fqdn2",
		},
	})
	rd.Set("gateway_names", []interface{}{
		map[string]interface{}{
			"name":            "name1",
			"tls_passthrough": false,
			"backends": []interface{}{
				map[string]interface{}{
					"address":   "add3",
					"namespace": "ns3",
					"password":  "pss3",
				},
				map[string]interface{}{
					"address":   "add4",
					"namespace": "ns4",
					"password":  "pss4",
				},
			},
			"fqdn": "fqdn1",
		},
		map[string]interface{}{
			"name":            "name2",
			"tls_passthrough": false,
			"backends": []interface{}{
				map[string]interface{}{
					"address":   "add3",
					"namespace": "ns3",
					"password":  "pss3",
				},
				map[string]interface{}{
					"address":   "add4",
					"namespace": "ns4",
					"password":  "pss4",
				},
			},
			"fqdn": "fqdn2",
		},
	})
	rd.Set("node_deployment_id", map[string]interface{}{
		"1": 2,
	})
	return rd
}

func getEmptyResourceDeployment() ResourceData {
	rd := ResourceData{}
	rd.data = map[string]interface{}{}
	rd.Set("id", "")
	rd.Set("node", 0)
	rd.Set("disks", []interface{}{})
	rd.Set("zdbs", []interface{}{})
	rd.Set("vms", []interface{}{})
	rd.Set("qsfss", []interface{}{})
	rd.Set("ip_range", "")
	rd.Set("network_name", "")
	rd.Set("fqdn", []interface{}{})
	rd.Set("gateway_names", "")
	rd.Set("node_deployment_id", map[string]interface{}{})
	return rd
}

type Obj struct {
	Name        string       `name:"name"`
	Value       int64        `name:"value"`
	ZdbBackends []ZdbBackend `name:"zdb_backends"`
}

func getFilledObj() ResourceData {
	rd := ResourceData{}
	rd.data = map[string]interface{}{}
	rd.Set("name", "mario")
	rd.Set("value", 100)
	rd.Set("zdb_backends", []interface{}{
		map[string]interface{}{
			"address":   "add1",
			"namespace": "ns1",
			"password":  "pass1",
		},
		map[string]interface{}{
			"address":   "add2",
			"namespace": "ns2",
			"password":  "pass2",
		},
	})

	return rd
}

func getEmptyObj() ResourceData {
	rd := ResourceData{}
	rd.data = map[string]interface{}{}
	rd.Set("name", "")
	rd.Set("value", 0)
	rd.Set("zdb_backends", []interface{}{})
	return rd
}

func TestConverter(t *testing.T) {

	dp := getDeployment()
	rd := getFilledResourceData()

	newDP := DeploymentDeployer{}
	newRD := getEmptyResourceDeployment()
	// dp := Obj{
	// 	Name:  "mario",
	// 	Value: int64(100),
	// 	ZdbBackends: []ZdbBackend{
	// 		{
	// 			Address:   "add1",
	// 			Namespace: "ns1",
	// 			Password:  "pass1",
	// 		},
	// 		{
	// 			Address:   "add2",
	// 			Namespace: "ns2",
	// 			Password:  "pass2",
	// 		},
	// 	},
	// }
	// newRD := getEmptyObj()
	// rd := getFilledObj()

	err := encode(dp, &newRD)
	if err != nil {
		log.Printf("error in encoding: %+v", err)
		assert.Equal(t, nil, err)
	}
	assert.Equal(t, rd, newRD)

	err2 := decode(&newDP, &rd)
	if err2 != nil {
		log.Printf("error in decoding: %+v", err2)
		assert.Equal(t, nil, err2)
	}
	assert.Equal(t, dp, newDP)

}
