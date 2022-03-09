package gridproxy

import (
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const NodeUP = "up"
const NodeDOWN = "down"

// capacityResult is the NodeData capacity results to unmarshal json in it
type capacityResult struct {
	Total gridtypes.Capacity `json:"total_resources"`
	Used  gridtypes.Capacity `json:"used_resources"`
}

// NodeInfo is node specific info, queried directly from the node
type NodeInfo struct {
	Capacity   capacityResult `json:"capacity"`
	DMI        dmi.DMI        `json:"dmi"`
	Hypervisor string         `json:"hypervisor"`
}

type PublicConfig struct {
	Domain string `json:"domain"`
	Gw4    string `json:"gw4"`
	Gw6    string `json:"gw6"`
	Ipv4   string `json:"ipv4"`
	Ipv6   string `json:"ipv6"`
}

type ErrorReply struct {
	Error string `json:"error"`
}

// Node is a struct holding the data for a node for the nodes view
type Node struct {
	Version           int                `json:"version"`
	ID                string             `json:"id"`
	NodeID            uint32             `json:"nodeId"`
	FarmID            int                `json:"farmId"`
	TwinID            int                `json:"twinId"`
	Country           string             `json:"country"`
	GridVersion       int                `json:"gridVersion"`
	City              string             `json:"city"`
	Uptime            int64              `json:"uptime"`
	Created           int64              `json:"created"`
	FarmingPolicyID   int                `json:"farmingPolicyId"`
	UpdatedAt         string             `json:"updatedAt"`
	TotalResources    gridtypes.Capacity `json:"total_resources"`
	UsedResources     gridtypes.Capacity `json:"used_resources"`
	Location          Location           `json:"location"`
	PublicConfig      PublicConfig       `json:"publicConfig"`
	Status            string             `json:"status"` // added node status field for up or down
	CertificationType string             `json:"certificationType"`
}
type Location struct {
	Country string `json:"country"`
	City    string `json:"city"`
}
type NodeStatus struct {
	Status string `json:"nodes"`
}

// Nodes is struct for the whole nodes view
type Nodes struct {
	Data []Node `json:"nodes"`
}

// NodeResponseStruct is struct for the whole nodes view
type NodesResponse struct {
	Nodes Nodes `json:"data"`
}

type NodeID struct {
	NodeID uint32 `json:"nodeId"`
}

// nodeIdData is the nodeIdData to unmarshal json in it
type NodeIDData struct {
	NodeResult []NodeID `json:"nodes"`
}

// nodeIdResult is the nodeIdResult  to unmarshal json in it
type NodeIDResult struct {
	Data NodeIDData `json:"data"`
}

type Farm struct {
	Name            string     `json:"name"`
	FarmID          int        `json:"farmId"`
	TwinID          int        `json:"twinId"`
	Version         int        `json:"version"`
	PricingPolicyID int        `json:"pricingPolicyId"`
	StellarAddress  string     `json:"stellarAddress"`
	PublicIps       []PublicIP `json:"publicIps"`
}
type PublicIP struct {
	ID         string `json:"id"`
	IP         string `json:"ip"`
	FarmID     string `json:"farmId"`
	ContractID int    `json:"contractId"`
	Gateway    string `json:"gateway"`
}

type FarmResult = []Farm

type farmDataV0 struct {
	Farms []Farm `json:"farms"`
}

// FarmResult is to unmarshal json in it
type FarmResultV0 struct {
	Data farmDataV0 `json:"data"`
}
type capacityResultV0 struct {
	Total gridtypes.Capacity `json:"total"`
	Used  gridtypes.Capacity `json:"used"`
}
type NodeInfoV0 struct {
	Capacity   capacityResultV0 `json:"capacity"`
	DMI        dmi.DMI          `json:"dmi"`
	Hypervisor string           `json:"hypervisor"`
}
