package state

import (
	"encoding/json"
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
	state := NewState()
	state.Networks["abc"] = NewNetwork()
	state.Networks["abc"].Subnets[15] = "10.1.1.0/24"
	state.Networks["abc"].NodeIPs[32] = make(DeploymentIPs)
	bt1, err := st.Marshal()
	assert.NoError(t, err)
	bt2, err := json.Marshal(state)
	assert.NoError(t, err)
	assert.Equal(t, bt1, bt2)
	assert.NoError(t, err)
}
