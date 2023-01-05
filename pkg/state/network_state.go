// Package state provides a state to save the user work in a database.
package state

// NetworkMap is a map of of names and their networks
type NetworkMap map[string]network

type network struct {
	Subnets               map[uint32]string     `json:"subnets"`
	NodeDeploymentHostIDs NodeDeploymentHostIDs `json:"node_ips"`
}

type NodeDeploymentHostIDs map[uint32]deploymentHostIDs

type deploymentHostIDs map[string][]byte

func NewNetwork() network {
	return network{
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
func (n *network) GetNodeSubnet(nodeID uint32) string {
	return n.Subnets[nodeID]
}

// SetNodeSubnet sets a node subnet with its ID and subnet
func (n *network) SetNodeSubnet(nodeID uint32, subnet string) {
	n.Subnets[nodeID] = subnet
}

// DeleteNodeSubnet deletes a node subnet using its ID
func (n *network) DeleteNodeSubnet(nodeID uint32) {
	delete(n.Subnets, nodeID)
}

func (n *network) GetUsedNetworkHostIDs(nodeID uint32) []byte {
	ips := []byte{}
	for _, v := range n.NodeDeploymentHostIDs[nodeID] {
		ips = append(ips, v...)
	}
	return ips
}

func (n *network) GetDeploymentHostIDs(nodeID uint32, deploymentID string) []byte {
	if n.NodeDeploymentHostIDs[nodeID] == nil {
		return []byte{}
	}
	return n.NodeDeploymentHostIDs[nodeID][deploymentID]
}

func (n *network) SetDeploymentHostIDs(nodeID uint32, deploymentID string, ips []byte) {
	if n.NodeDeploymentHostIDs[nodeID] == nil {
		n.NodeDeploymentHostIDs[nodeID] = deploymentHostIDs{}
	}
	n.NodeDeploymentHostIDs[nodeID][deploymentID] = ips
}

func (n *network) DeleteDeployment(nodeID uint32, deploymentID string) {
	if n.NodeDeploymentHostIDs[nodeID] == nil {
		return
	}
	delete(n.NodeDeploymentHostIDs[nodeID], deploymentID)
}
