// Package state provides a state to save the user work in a database.
package state

// NetworkState is a map of of names and their networks
type NetworkState map[string]Network

// Network struct includes subnets and node IPs
type Network struct {
	Subnets               map[uint32]string     `json:"subnets"`
	NodeDeploymentHostIDs NodeDeploymentHostIDs `json:"node_ips"`
}

// NodeDeploymentHostIDs is a map for nodes ID and its deployments' IPs
type NodeDeploymentHostIDs map[uint32]DeploymentHostIDs

// DeploymentHostIDs is a map for deployment and its IPs
type DeploymentHostIDs map[string][]byte

// NewNetwork creates a new Network
func NewNetwork() Network {
	return Network{
		Subnets:               map[uint32]string{},
		NodeDeploymentHostIDs: NodeDeploymentHostIDs{},
	}
}

// GetNetwork get a network using its name
func (nm NetworkState) GetNetwork(networkName string) Network {
	if _, ok := nm[networkName]; !ok {
		nm[networkName] = NewNetwork()
	}
	net := nm[networkName]
	return net
}

// DeleteNetwork deletes a network using its name
func (nm NetworkState) DeleteNetwork(networkName string) {
	delete(nm, networkName)
}

// GetNodeSubnet gets a node subnet using its ID
func (n *Network) GetNodeSubnet(nodeID uint32) string {
	return n.Subnets[nodeID]
}

// SetNodeSubnet sets a node subnet with its ID and subnet
func (n *Network) SetNodeSubnet(nodeID uint32, subnet string) {
	n.Subnets[nodeID] = subnet
}

// DeleteNodeSubnet deletes a node subnet using its ID
func (n *Network) DeleteNodeSubnet(nodeID uint32) {
	delete(n.Subnets, nodeID)
}

// GetUsedNetworkHostIDs gets the used host IDs on the overlay network
func (n *Network) GetUsedNetworkHostIDs(nodeID uint32) []byte {
	ips := []byte{}
	for _, v := range n.NodeDeploymentHostIDs[nodeID] {
		ips = append(ips, v...)
	}
	return ips
}

// GetDeploymentHostIDs gets the private network host IDs relevant to the deployment
func (n *Network) GetDeploymentHostIDs(nodeID uint32, deploymentID string) []byte {
	if n.NodeDeploymentHostIDs[nodeID] == nil {
		return []byte{}
	}
	return n.NodeDeploymentHostIDs[nodeID][deploymentID]
}

// SetDeploymentHostIDs sets the relevant deployment host IDs
func (n *Network) SetDeploymentHostIDs(nodeID uint32, deploymentID string, ips []byte) {
	if n.NodeDeploymentHostIDs[nodeID] == nil {
		n.NodeDeploymentHostIDs[nodeID] = DeploymentHostIDs{}
	}
	n.NodeDeploymentHostIDs[nodeID][deploymentID] = ips
}

// DeleteDeploymentHostIDs deletes a deployment host IDs
func (n *Network) DeleteDeploymentHostIDs(nodeID uint32, deploymentID string) {
	delete(n.NodeDeploymentHostIDs[nodeID], deploymentID)
}
