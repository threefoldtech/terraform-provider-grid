package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestState(t *testing.T) {
	f, err := NewLocalStateDB(TypeFile)
	assert.NoError(t, err)
	err = f.Load()
	assert.NoError(t, err)
	st := f.GetState()
	ns := st.GetNetworkState()
	network := ns.GetNetwork("abc")
	network.SetNodeSubnet(32, "10.1.1.0/24")
	network.SetNodeSubnet(15, "10.1.1.0/24")
	network.SetDeploymentIPs(32, "12345", []byte{1, 2, 3})
	network.DeleteDeployment(32, "12345")
	network.DeleteNodeSubnet(32)
	err = f.Save()
	assert.NoError(t, err)
}
