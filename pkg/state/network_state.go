// Package state provides a state to save the user work in a database.
package state

// NetworkMap is a map of of names and their networks
type NetworkMap map[string]Network

// Network struct includes subnets and node IPs
type Network struct {
	Subnets               map[uint32]string     `json:"subnets"`
	NodeDeploymentHostIDs NodeDeploymentHostIDs `json:"node_ips"`
}

// NodeDeploymentHostIDs is a map for nodes ID and its deployments' host IDs for IPs
type NodeDeploymentHostIDs map[uint32]DeploymentHostIDs

// DeploymentHostIDs is a map for deployment and its host IDs for IPs
type DeploymentHostIDs map[string][]byte

// NewNetwork generates a new network
func NewNetwork() Network {
	return Network{
		Subnets:               map[uint32]string{},
		NodeDeploymentHostIDs: NodeDeploymentHostIDs{},
	}
}

// GetNetwork get a network using its name
func (nm NetworkMap) GetNetwork(networkName string) NetworkInterface {
	if _, ok := nm[networkName]; !ok {
		nm[networkName] = NewNetwork()
	}
	net := nm[networkName]
	return &net
}

// DeleteNetwork deletes a network using its name
func (nm NetworkMap) DeleteNetwork(networkName string) {
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

// GetNodeIPsList returns a list of IPs for a node given its ID
func (n *Network) GetNodeIPsList(nodeID uint32) []byte {
	ips := []byte{}
	for _, v := range n.NodeDeploymentHostIDs[nodeID] {
		ips = append(ips, v...)
	}
	return ips
}

// GetDeploymentIPs returns a list of IPs for a deployment on a given node
func (n *Network) GetDeploymentIPs(nodeID uint32, deploymentID string) []byte {
	if n.NodeDeploymentHostIDs[nodeID] == nil {
		return []byte{}
	}
	return n.NodeDeploymentHostIDs[nodeID][deploymentID]
}

// SetDeploymentIPs sets a list of IPs for a deployment on a given node
func (n *Network) SetDeploymentIPs(nodeID uint32, deploymentID string, ips []byte) {
	if n.NodeDeploymentHostIDs[nodeID] == nil {
		n.NodeDeploymentHostIDs[nodeID] = DeploymentHostIDs{}
	}
	n.NodeDeploymentHostIDs[nodeID][deploymentID] = ips
}

// DeleteDeployment deletes a deployment on a node
func (n *Network) DeleteDeployment(nodeID uint32, deploymentID string) {
	delete(n.NodeDeploymentHostIDs[nodeID], deploymentID)
}
