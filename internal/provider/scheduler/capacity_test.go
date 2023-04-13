package scheduler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	proxyTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

var (
	node = proxyTypes.Node{
		UsedResources: proxyTypes.Capacity{
			HRU: 1,
			SRU: 2,
			MRU: 3,
		},
		TotalResources: proxyTypes.Capacity{
			HRU: 4,
			SRU: 5,
			MRU: 6,
		},
	}
)

func TestFreeCapacity(t *testing.T) {
	cap := freeCapacity(&node)
	assert.Equal(t, cap.HRU, uint64(3), "hru")
	assert.Equal(t, cap.SRU, uint64(3), "sru")
	assert.Equal(t, cap.MRU, uint64(3), "mru")
}
