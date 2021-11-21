package rmbproxytypes

import (
	"github.com/gomodule/redigo/redis"
	"github.com/patrickmn/go-cache"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/rmb"
)

// DefaultExplorerURL is the default explorer graphql url
const DefaultExplorerURL string = "https://tfchain.dev.threefold.io/graphql/graphql"

// App is the main app objects
type App struct {
	explorer string
	redis    *redis.Pool
	rmb      rmb.Client
	lruCache *cache.Cache
}

// OffsetKey is the type holds the request context
type offsetKey struct{}

// SpecificFarmKey is the type holds the request context
type specificFarmKey struct{}

// MaxResultKey is the type holds the request context
type maxResultKey struct{}

// isGatewayKey is the type holds the request context
type isGatewayKey struct{}

// NodeTwinID is the node twin ID to unmarshal json in it
type nodeTwinID struct {
	TwinID uint32 `json:"twinId"`
}

// NodeData is having nodeTwinID to unmarshal json in it
type nodeData struct {
	NodeResult []nodeTwinID `json:"nodes"`
}

// NodeResult is the NodeData to unmarshal nodeTwinID json in it
type nodeResult struct {
	Data nodeData `json:"data"`
}

// CapacityResult is the NodeData capacity results to unmarshal json in it
type capacityResult struct {
	Total gridtypes.Capacity `json:"total"`
	Used  gridtypes.Capacity `json:"used"`
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

// Node is a struct holding the data for a node for the nodes view
type Node struct {
	Version         int          `json:"version"`
	ID              string       `json:"id"`
	NodeID          int          `json:"nodeId"`
	FarmID          int          `json:"farmId"`
	TwinID          int          `json:"twinId"`
	Country         string       `json:"country"`
	GridVersion     int          `json:"gridVersion"`
	City            string       `json:"city"`
	Uptime          int64        `json:"uptime"`
	Created         int64        `json:"created"`
	FarmingPolicyID int          `json:"farmingPolicyId"`
	UpdatedAt       string       `json:"updatedAt"`
	Cru             string       `json:"cru"`
	Mru             string       `json:"mru"`
	Sru             string       `json:"sru"`
	Hru             string       `json:"hru"`
	PublicConfig    PublicConfig `json:"publicConfig"`
	State           string       `json:"state"` // added node state field for up or down
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
	Name            string `json:"name"`
	FarmID          int    `json:"farmId"`
	TwinID          int    `json:"twinId"`
	Version         int    `json:"version"`
	PricingPolicyID int    `json:"pricingPolicyId"`
	StellarAddress  string `json:"stellarAddress"`
}

type PublicIP struct {
	ID         string `json:"id"`
	IP         string `json:"ip"`
	FarmID     string `json:"farmId"`
	ContractID int    `json:"contractId"`
	Gateway    string `json:"gateway"`
}

type farmData struct {
	Farms     []Farm     `json:"farms"`
	PublicIps []PublicIP `json:"publicIps"`
}

// FarmResult is to unmarshal json in it
type FarmResult struct {
	Data farmData `json:"data"`
}
