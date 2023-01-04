// Package workloads includes workloads types (vm, zdb, qsfs, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/hex"
	"log"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// QSFS struct
type QSFS struct {
	Name                 string
	Description          string
	Cache                int
	MinimalShards        uint32
	ExpectedShards       uint32
	RedundantGroups      uint32
	RedundantNodes       uint32
	MaxZDBDataDirSize    uint32
	EncryptionAlgorithm  string
	EncryptionKey        string
	CompressionAlgorithm string
	Metadata             Metadata
	Groups               Groups

	MetricsEndpoint string
}

// Metadata for QSFS
type Metadata struct {
	Type                string
	Prefix              string
	EncryptionAlgorithm string
	EncryptionKey       string
	Backends            Backends
}

// Group is a zos group
type Group struct {
	Backends Backends
}

// Backend is a zos backend
type Backend zos.ZdbBackend

// Groups is a list of groups
type Groups []Group

// Backends is a list of backends
type Backends []Backend

func (g *Group) zosGroup() zos.ZdbGroup {
	z := zos.ZdbGroup{
		Backends: make([]zos.ZdbBackend, 0),
	}
	for _, b := range g.Backends {
		z.Backends = append(z.Backends, b.zosBackend())
	}
	return z
}

func (gs *Groups) zosGroups() []zos.ZdbGroup {
	z := make([]zos.ZdbGroup, 0)
	for _, e := range *gs {
		z = append(z, e.zosGroup())
	}
	return z
}

func (b *Backend) zosBackend() zos.ZdbBackend {
	return zos.ZdbBackend(*b)
}

func (bs *Backends) zosBackends() []zos.ZdbBackend {
	z := make([]zos.ZdbBackend, 0)
	for _, e := range *bs {
		z = append(z, e.zosBackend())
	}
	return z
}

// BackendsFromZos gets backends from zos
func BackendsFromZos(bs []zos.ZdbBackend) Backends {
	z := make(Backends, 0)
	for _, e := range bs {
		z = append(z, Backend(e))
	}
	return z
}

// GroupsFromZos gets groups from zos
func GroupsFromZos(gs []zos.ZdbGroup) Groups {
	z := make(Groups, 0)
	for _, e := range gs {
		z = append(z, Group{
			Backends: BackendsFromZos(e.Backends),
		})
	}
	return z
}

// Dictify converts a metadata to a map
func (m *Metadata) Dictify() map[string]interface{} {
	res := make(map[string]interface{})
	res["type"] = m.Type
	res["prefix"] = m.Prefix
	res["encryption_algorithm"] = m.EncryptionAlgorithm
	res["encryption_key"] = m.EncryptionKey
	res["backends"] = m.Backends.Listify()
	return res
}

// Dictify converts a group data to a map
func (g *Group) Dictify() map[string]interface{} {
	res := make(map[string]interface{})
	res["backends"] = g.Backends.Listify()
	return res
}

// Dictify converts a backend data to a map
func (b *Backend) Dictify() map[string]interface{} {
	res := make(map[string]interface{})
	res["address"] = b.Address
	res["namespace"] = b.Namespace
	res["password"] = b.Password
	return res
}

// Listify lists the backends
func (bs *Backends) Listify() []interface{} {
	res := make([]interface{}, 0)
	for _, b := range *bs {
		res = append(res, b.Dictify())
	}
	return res
}

// Listify lists the groups
func (gs *Groups) Listify() []interface{} {
	res := make([]interface{}, 0)
	for _, g := range *gs {
		res = append(res, g.Dictify())
	}
	return res
}

// NewQSFSFromSchema generates a new QSFS from a given map of its data
func NewQSFSFromSchema(qsfs map[string]interface{}) QSFS {
	metadataIf := qsfs["metadata"].([]interface{})
	metadataMap := metadataIf[0].(map[string]interface{})

	metadata := Metadata{
		Type:                metadataMap["type"].(string),
		Prefix:              metadataMap["prefix"].(string),
		EncryptionAlgorithm: metadataMap["encryption_algorithm"].(string),
		EncryptionKey:       metadataMap["encryption_key"].(string),
		Backends:            getBackends(metadataMap["backends"].([]interface{})),
	}
	groupsIf := qsfs["groups"].([]interface{})
	groups := make([]Group, 0, len(groupsIf))
	for _, gr := range groupsIf {
		groupMap := gr.(map[string]interface{})
		groups = append(groups, Group{
			Backends: getBackends(groupMap["backends"].([]interface{})),
		})
	}
	return QSFS{
		Name:                 qsfs["name"].(string),
		Description:          qsfs["description"].(string),
		Cache:                qsfs["cache"].(int),
		MinimalShards:        uint32(qsfs["minimal_shards"].(int)),
		ExpectedShards:       uint32(qsfs["expected_shards"].(int)),
		RedundantGroups:      uint32(qsfs["redundant_groups"].(int)),
		RedundantNodes:       uint32(qsfs["redundant_nodes"].(int)),
		MaxZDBDataDirSize:    uint32(qsfs["max_zdb_data_dir_size"].(int)),
		EncryptionAlgorithm:  qsfs["encryption_algorithm"].(string),
		EncryptionKey:        qsfs["encryption_key"].(string),
		CompressionAlgorithm: qsfs["compression_algorithm"].(string),
		Metadata:             metadata,
		Groups:               groups,
	}
}

// NewQSFSFromWorkload generates a new QSFS from a workload
func NewQSFSFromWorkload(wl *gridtypes.Workload) (QSFS, error) {

	var data *zos.QuantumSafeFS
	wd, err := wl.WorkloadData()
	if err != nil {
		return QSFS{}, err
	}
	var res zos.QuatumSafeFSResult
	if err := wl.Result.Unmarshal(&res); err != nil {
		return QSFS{}, err
	}
	log.Printf("wl.Result.unm: %s %s\n", res.MetricsEndpoint, res.Path)
	data = wd.(*zos.QuantumSafeFS)
	return QSFS{
		Name:                 string(wl.Name),
		Description:          string(wl.Description),
		Cache:                int(data.Cache) / int(gridtypes.Megabyte),
		MinimalShards:        data.Config.MinimalShards,
		ExpectedShards:       data.Config.ExpectedShards,
		RedundantGroups:      data.Config.RedundantGroups,
		RedundantNodes:       data.Config.RedundantNodes,
		MaxZDBDataDirSize:    data.Config.MaxZDBDataDirSize,
		EncryptionAlgorithm:  string(data.Config.Encryption.Algorithm),
		EncryptionKey:        hex.EncodeToString(data.Config.Encryption.Key),
		CompressionAlgorithm: data.Config.Compression.Algorithm,
		Metadata: Metadata{
			Type:                data.Config.Meta.Type,
			Prefix:              data.Config.Meta.Config.Prefix,
			EncryptionAlgorithm: string(data.Config.Meta.Config.Encryption.Algorithm),
			EncryptionKey:       hex.EncodeToString(data.Config.Meta.Config.Encryption.Key),
			Backends:            BackendsFromZos(data.Config.Meta.Config.Backends),
		},
		Groups:          GroupsFromZos(data.Config.Groups),
		MetricsEndpoint: res.MetricsEndpoint,
	}, nil
}

func getBackends(backendsIf []interface{}) []Backend {
	backends := make([]Backend, 0, len(backendsIf))
	for _, b := range backendsIf {
		backendMap := b.(map[string]interface{})
		backends = append(backends, Backend{
			Address:   backendMap["address"].(string),
			Password:  backendMap["password"].(string),
			Namespace: backendMap["namespace"].(string),
		})
	}
	return backends
}

// ZosWorkload generates a zos workload
func (q *QSFS) ZosWorkload() (gridtypes.Workload, error) {
	k, err := hex.DecodeString(q.EncryptionKey)
	if err != nil {
		return gridtypes.Workload{}, err
	}
	mk, err := hex.DecodeString(q.EncryptionKey)
	if err != nil {
		return gridtypes.Workload{}, err
	}
	workload := gridtypes.Workload{
		Version:     0,
		Name:        gridtypes.Name(q.Name),
		Type:        zos.QuantumSafeFSType,
		Description: q.Description,
		Data: gridtypes.MustMarshal(zos.QuantumSafeFS{
			Cache: gridtypes.Unit(uint64(q.Cache) * uint64(gridtypes.Megabyte)),
			Config: zos.QuantumSafeFSConfig{
				MinimalShards:     q.MinimalShards,
				ExpectedShards:    q.ExpectedShards,
				RedundantGroups:   q.RedundantGroups,
				RedundantNodes:    q.RedundantNodes,
				MaxZDBDataDirSize: q.MaxZDBDataDirSize,
				Encryption: zos.Encryption{
					Algorithm: zos.EncryptionAlgorithm(q.EncryptionAlgorithm),
					Key:       zos.EncryptionKey(k),
				},
				Meta: zos.QuantumSafeMeta{
					Type: q.Metadata.Type,
					Config: zos.QuantumSafeConfig{
						Prefix: q.Metadata.Prefix,
						Encryption: zos.Encryption{
							Algorithm: zos.EncryptionAlgorithm(q.EncryptionAlgorithm),
							Key:       zos.EncryptionKey(mk),
						},
						Backends: q.Metadata.Backends.zosBackends(),
					},
				},
				Groups: q.Groups.zosGroups(),
				Compression: zos.QuantumCompression{
					Algorithm: q.CompressionAlgorithm,
				},
			},
		}),
	}

	return workload, nil
}

// QSFS returns the name of QSFS
func (q *QSFS) QSFS() string {
	return q.Name
}

// UpdateFromWorkload updates a QSFS from a workload
// TODO: no updates, should construct itself from the workload
func (q *QSFS) UpdateFromWorkload(wl *gridtypes.Workload) error {
	if wl == nil {
		q.MetricsEndpoint = ""
		return nil
	}
	var res zos.QuatumSafeFSResult
	if err := wl.Result.Unmarshal(&res); err != nil {
		return errors.Wrap(err, "error unmarshalling json")

	}
	q.MetricsEndpoint = res.MetricsEndpoint
	return nil
}

// Dictify converts a QSFS data to a map
func (q *QSFS) Dictify() map[string]interface{} {
	res := make(map[string]interface{})
	res["name"] = q.Name
	res["description"] = q.Description
	res["cache"] = q.Cache
	res["minimal_shards"] = q.MinimalShards
	res["expected_shards"] = q.ExpectedShards
	res["redundant_groups"] = q.RedundantGroups
	res["redundant_nodes"] = q.RedundantNodes
	res["max_zdb_data_dir_size"] = q.MaxZDBDataDirSize
	res["encryption_algorithm"] = q.EncryptionAlgorithm
	res["encryption_key"] = q.EncryptionKey
	res["compression_algorithm"] = q.CompressionAlgorithm
	res["metrics_endpoint"] = q.MetricsEndpoint
	res["metadata"] = []interface{}{q.Metadata.Dictify()}
	res["groups"] = q.Groups.Listify()
	return res
}
