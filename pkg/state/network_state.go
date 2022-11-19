package state

type networkingState map[string]network

type network struct {
	Subnets map[uint32]string `json:"subnets"`
	NodeIPs nodeIPs           `json:"node_ips"`
}

type nodeIPs map[uint32]deploymentIPs

type deploymentIPs map[string][]byte

func NewNetwork() network {
	return network{
		Subnets: map[uint32]string{},
		NodeIPs: nodeIPs{},
	}
}

func (ns networkingState) GetNetwork(networkName string) Network {
	if _, ok := ns[networkName]; !ok {
		ns[networkName] = NewNetwork()
	}
	net := ns[networkName]
	return &net
}

func (ns networkingState) DeleteNetwork(networkName string) {
	delete(ns, networkName)
}

func (n *network) GetNodeSubnet(nodeID uint32) string {
	return n.Subnets[nodeID]
}

func (n *network) SetNodeSubnet(nodeID uint32, subnet string) {
	n.Subnets[nodeID] = subnet
}

func (n *network) DeleteNodeSubnet(nodeID uint32) {
	delete(n.Subnets, nodeID)
}

func (n *network) GetNodeIPsList(nodeID uint32) []byte {
	ips := []byte{}
	for _, v := range n.NodeIPs[nodeID] {
		ips = append(ips, v...)
	}
	return ips
}

func (n *network) GetDeploymentIPs(nodeID uint32, deploymentID string) []byte {
	if n.NodeIPs[nodeID] == nil {
		return []byte{}
	}
	return n.NodeIPs[nodeID][deploymentID]
}

func (n *network) SetDeploymentIPs(nodeID uint32, deploymentID string, ips []byte) {
	if n.NodeIPs[nodeID] == nil {
		n.NodeIPs[nodeID] = deploymentIPs{}
	}
	n.NodeIPs[nodeID][deploymentID] = ips
}

func (n *network) DeleteDeployment(nodeID uint32, deploymentID string) {
	if n.NodeIPs[nodeID] == nil {
		return
	}
	delete(n.NodeIPs[nodeID], deploymentID)
}

func (n *network) GetSubnets() map[uint32]string {
	return n.Subnets
}

func (n *network) GetNodeIPs() nodeIPs {
	return n.NodeIPs
}
