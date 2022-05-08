package gridproxy

import (
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const NodeUP = "up"
const NodeDOWN = "down"

// capacityResult is the NodeData capacity results to unmarshal json in it
type CapacityResult struct {
	Total gridtypes.Capacity `json:"total_resources"`
	Used  gridtypes.Capacity `json:"used_resources"`
}

// NodeInfo is node specific info, queried directly from the node
type NodeInfo struct {
	Capacity CapacityResult `json:"capacity"`
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
	Status string `json:"status"`
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

// FarmResult is to unmarshal json in it

// Limit used for pagination
type Limit struct {
	Size     uint64
	Page     uint64
	RetCount bool
}

// NodeFilter node filters
type NodeFilter struct {
	Status       *string
	FreeMRU      *uint64
	FreeHRU      *uint64
	FreeSRU      *uint64
	Country      *string
	City         *string
	FarmName     *string
	FarmIDs      []uint64
	FreeIPs      *uint64
	IPv4         *bool
	IPv6         *bool
	Domain       *bool
	Rentable     *bool
	RentedBy     *uint64
	AvailableFor *uint64
}

// FarmFilter farm filters
type FarmFilter struct {
	FreeIPs           *uint64
	TotalIPs          *uint64
	StellarAddress    *string
	PricingPolicyID   *uint64
	Version           *uint64
	FarmID            *uint64
	TwinID            *uint64
	Name              *string
	NameContains      *string
	CertificationType *string
	Dedicated         *bool
}
