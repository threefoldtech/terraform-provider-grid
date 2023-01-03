// package workloads includes workloads types (vm, zdb, qsfs, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func zdbInterface() map[string]interface{} {
	return map[string]interface{}{
		"name":        "z",
		"password":    "pass",
		"public":      true,
		"size":        1024,
		"description": "des",
		"mode":        "user",
		"ips": []interface{}{
			"::1",
			"::2",
		},
		"namespace": "ns1",
		"port":      9090,
	}
}
func zdbObj() *ZDB {
	return &ZDB{
		Name:        "z",
		Password:    "pass",
		Public:      true,
		Size:        1024,
		Description: "des",
		Mode:        "user",
		IPs:         []string{"::1", "::2"},
		Namespace:   "ns1",
		Port:        9090,
	}
}
func zdbWl(zdb *ZDB) gridtypes.Workload {
	return gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name(zdb.Name),
		Type:    zos.ZDBType,
		Data: gridtypes.MustMarshal(zos.ZDB{
			Size:     gridtypes.Unit(zdb.Size * int(gridtypes.Gigabyte)),
			Mode:     zos.ZDBMode(zdb.Mode),
			Password: zdb.Password,
			Public:   zdb.Public,
		}),
		Metadata:    "",
		Description: zdb.Description,
	}
}

func TestGetZDBData(t *testing.T) {
	zdb := GetZdbData(zdbInterface())
	obj := zdbObj()
	assert.Equal(t, zdb, *obj)
}
func TestGenerateZDBWorkload(t *testing.T) {
	zdb := zdbObj()
	wl := zdbWl(zdb)
	assert.Equal(t, zdb.GenerateZDBWorkload(), wl)
}
func TestNewZDBFromWorkload(t *testing.T) {

}
