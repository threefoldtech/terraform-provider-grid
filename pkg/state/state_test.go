package state

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestState(t *testing.T) {
	var db DB

	var localState StateI
	var newState State

	var marshalLocalState []byte
	var marshalNewState []byte

	var err error

	t.Run("test_local_db", func(t *testing.T) {
		db, err = NewLocalStateDB(TypeFile)
		assert.NoError(t, err)

		err = db.Load()
		assert.NoError(t, err)
	})

	t.Run("test_get_local_state", func(t *testing.T) {
		localState = db.GetState()

		networkState := localState.GetNetworkState()
		network := networkState.GetNetwork("abc")
		network.SetNodeSubnet(32, "10.1.1.0/24")
		network.SetNodeSubnet(15, "10.1.1.0/24")
		network.SetDeploymentHostIDs(32, "12345", []byte{1, 2, 3})
		network.DeleteDeployment(32, "12345")
		network.DeleteNodeSubnet(32)

		err = db.Save()
		assert.NoError(t, err)
	})

	t.Run("test_marshal_local_state", func(t *testing.T) {
		marshalLocalState, err = json.Marshal(localState)
		assert.NoError(t, err)
	})

	t.Run("test_new_state", func(t *testing.T) {
		newState = NewState()

		newState.Networks["abc"] = NewNetwork()
		newState.Networks["abc"].Subnets[15] = "10.1.1.0/24"
		newState.Networks["abc"].NodeDeploymentHostIDs[32] = make(deploymentHostIDs)
	})

	t.Run("test_marshal_local_state", func(t *testing.T) {
		marshalNewState, err = json.Marshal(newState)
		assert.NoError(t, err)
	})

	t.Run("test_equal_local_and_created_state", func(t *testing.T) {
		assert.Equal(t, marshalLocalState, marshalNewState)
	})

	assert.NoError(t, err)
}
