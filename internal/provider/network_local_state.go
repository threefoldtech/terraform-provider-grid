package provider

import (
	"encoding/json"
	// "io/ioutil"

	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type LocalNetworkState map[string]NetworkState

type NetworkState struct {
	NodesSubnets map[uint32]string            `json:"nodes_subnets"`
	NodesUsedIPs map[uint32]DeploymentUsedIPs `json:"nodes_used_ips"`
}

type DeploymentUsedIPs map[string][]byte

const FILE_NAME = "network_state.json"

func getNetworkLocalState() (LocalNetworkState, error) {
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

func (l LocalNetworkState) saveLocalState() error {
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

func (l LocalNetworkState) appendUsedIP(ipStr string, d *DeploymentDeployer) {
	networkState := l[d.NetworkName]

	if networkState.NodesUsedIPs == nil {
		networkState.NodesUsedIPs = map[uint32]DeploymentUsedIPs{}
	}
	deploymentUsedIPs := networkState.NodesUsedIPs[d.Node]
	if deploymentUsedIPs == nil {
		deploymentUsedIPs = map[string][]byte{}
	}
	ip := net.ParseIP(ipStr).To4()
	deploymentUsedIPs[d.Id] = append(deploymentUsedIPs[d.Id], ip[3])
	networkState.NodesUsedIPs[d.Node] = deploymentUsedIPs
	l[d.NetworkName] = networkState
}

func (n *NetworkState) getNodeUsedIPs(nodeID uint32) []byte {
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

func (n *NetworkState) removeDeploymentUsedIPs(nodeID uint32, deploymentID string) {
	if n.NodesUsedIPs == nil {
		n.NodesUsedIPs = map[uint32]DeploymentUsedIPs{}
	}
	if n.NodesUsedIPs[nodeID] == nil {
		n.NodesUsedIPs[nodeID] = DeploymentUsedIPs{}
	}
	delete(n.NodesUsedIPs[nodeID], deploymentID)
}

func (n *NetworkState) updateNodesSubnets(ranges map[uint32]gridtypes.IPNet) {
	if n.NodesSubnets == nil {
		n.NodesSubnets = map[uint32]string{}
	}
	for node, subnet := range ranges {
		n.NodesSubnets[node] = subnet.String()
	}
}

func deleteNetworkLocalState() error {
	return os.Remove(FILE_NAME)
}
