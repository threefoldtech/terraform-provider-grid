package state

// NetworkMap is a map of of names and their networks
type NetworkMap map[string]Network

// Network struct includes subnets and node IPs
type Network struct {
	Subnets map[uint32]string `json:"subnets"`
	NodeIPs NodeIPs           `json:"node_ips"`
}

// NodeIPs is a map for nodes ID and its deployments' IPs
type NodeIPs map[uint32]DeploymentIPs

// DeploymentIPs is a map for deployment and its IPs
type DeploymentIPs map[string][]byte

// NewNetwork generates a new network
func NewNetwork() Network {
	return Network{
		Subnets: map[uint32]string{},
		NodeIPs: NodeIPs{},
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
	for _, v := range n.NodeIPs[nodeID] {
		ips = append(ips, v...)
	}
	return ips
}

// GetDeploymentIPs returns a list of IPs for a deployment on a given node
func (n *Network) GetDeploymentIPs(nodeID uint32, deploymentID string) []byte {
	if n.NodeIPs[nodeID] == nil {
		return []byte{}
	}
	return n.NodeIPs[nodeID][deploymentID]
}

// SetDeploymentIPs sets a list of IPs for a deployment on a given node
func (n *Network) SetDeploymentIPs(nodeID uint32, deploymentID string, ips []byte) {
	if n.NodeIPs[nodeID] == nil {
		n.NodeIPs[nodeID] = DeploymentIPs{}
	}
	n.NodeIPs[nodeID][deploymentID] = ips
}

// DeleteDeployment deletes a deployment on a node
func (n *Network) DeleteDeployment(nodeID uint32, deploymentID string) {
	delete(n.NodeIPs[nodeID], deploymentID)
}
