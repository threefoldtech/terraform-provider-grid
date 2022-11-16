package local_state

import (
	"encoding/json"
	// "io/ioutil"

	"os"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type LocalNetworkState map[string]NetworkState

type NodesUsedIPs map[uint32]DeploymentUsedIPs

type NetworkState struct {
	NodesSubnets map[uint32]string `json:"nodes_subnets"`
	NodesUsedIPs NodesUsedIPs      `json:"nodes_used_ips"`
}

type DeploymentUsedIPs map[string][]byte

const FILE_NAME = "network_state.json"

func GetNetworkLocalState() (LocalNetworkState, error) {
	localNetworkState := LocalNetworkState{}
	_, err := os.Stat(FILE_NAME)
	if err != nil {
		return LocalNetworkState{}, err
	}
	content, err := os.ReadFile(FILE_NAME)
	if err != nil {
		return LocalNetworkState{}, nil
	}

	err = json.Unmarshal(content, &localNetworkState)
	if err != nil {
		return LocalNetworkState{}, nil
	}
	return localNetworkState, nil
}

func (l LocalNetworkState) GetNetworkState(networkName string) NetworkState {
	state := l[networkName]
	if state.NodesSubnets == nil {
		state.NodesSubnets = map[uint32]string{}
	}
	if state.NodesUsedIPs == nil {
		state.NodesUsedIPs = NodesUsedIPs{}
	}
	l[networkName] = state
	return state
}

func (l LocalNetworkState) SaveLocalState() error {
	if content, err := json.Marshal(&l); err == nil {
		err = os.WriteFile(FILE_NAME, content, 0644)
		if err != nil {
			return errors.Wrapf(err, "failed to write file: %s", FILE_NAME)
		}
	} else {
		return errors.Wrapf(err, "failed to save file: %s", FILE_NAME)
	}
	return nil
}

func (n *NetworkState) AccumulateNodeUsedIPs(nodeID uint32) []byte {
	usedIPs := []byte{}
	for node, deploymentUsedIPs := range n.NodesUsedIPs {
		if node == nodeID {
			for _, ips := range deploymentUsedIPs {
				usedIPs = append(usedIPs, ips...)
			}
		}
	}
	return usedIPs
}

func (n *NetworkState) GetDeploymentUsedIPs(nodeID uint32) DeploymentUsedIPs {
	if n.NodesUsedIPs[nodeID] == nil {
		n.NodesUsedIPs[nodeID] = DeploymentUsedIPs{}
	}
	return n.NodesUsedIPs[nodeID]
}

func (n *NetworkState) SetDeploymentUsedIPs(nodeID uint32, d DeploymentUsedIPs) {
	n.NodesUsedIPs[nodeID] = d
}

func (n *NetworkState) GetNodeSubnet(nodeID uint32) string {
	return n.NodesSubnets[nodeID]
}

func (n *NetworkState) RemoveDeploymentUsedIPs(nodeID uint32, deploymentID string) {
	deploymentUsedIPs := n.GetDeploymentUsedIPs(nodeID)
	delete(deploymentUsedIPs, deploymentID)
	n.NodesUsedIPs[nodeID] = deploymentUsedIPs
}

func (n *NetworkState) UpdateNodesSubnets(ranges map[uint32]gridtypes.IPNet) {
	for node, subnet := range ranges {
		n.NodesSubnets[node] = subnet.String()
	}
}

func DeleteNetworkLocalState() error {
	return os.Remove(FILE_NAME)
}
