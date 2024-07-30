package scheduler

import (
	"context"
	"fmt"
	"log"

	"github.com/pkg/errors"
)

const (
	rmbTimeout              = 40
	FarmerBotVersionAction  = "farmerbot.farmmanager.version"
	FarmerBotFindNodeAction = "farmerbot.nodemanager.findnode"
)

func (s *Scheduler) hasFarmerBot(ctx context.Context, farmID uint32) bool {
	ctx, cancel := context.WithTimeout(ctx, rmbTimeout)
	defer cancel()

	info, err := s.getFarmInfo(ctx, farmID)
	if err != nil {
		return false
	}

	dst := info.farmerTwinID

	service := fmt.Sprintf("farmerbot-%d", farmID)
	var version string
	err = s.rmbClient.CallWithSession(ctx, info.farmerTwinID, &service, FarmerBotVersionAction, nil, &version)
	if err != nil {
		log.Printf("error while pinging farmerbot on farm %d with farmer twin %d. %s", farmID, dst, err.Error())
	}

	return err == nil
}

func (n *Scheduler) farmerBotSchedule(ctx context.Context, r *Request) (uint32, error) {
	ctx, cancel := context.WithTimeout(ctx, rmbTimeout)
	defer cancel()

	info, err := n.getFarmInfo(ctx, r.FarmID)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get farm %d info", r.FarmID)
	}

	data := buildNodeOptions(r)
	var nodeID uint32

	service := fmt.Sprintf("farmerbot-%d", r.FarmID)
	if err := n.rmbClient.CallWithSession(ctx, info.farmerTwinID, &service, FarmerBotFindNodeAction, data, &nodeID); err != nil {
		return 0, err
	}

	log.Printf("got a node with id %d", nodeID)
	return nodeID, nil
}

type NodeFilterOption struct {
	NodesExcluded []uint32 `json:"nodes_excluded,omitempty"`
	Certified     bool     `json:"certified,omitempty"`
	Dedicated     bool     `json:"dedicated,omitempty"`
	PublicConfig  bool     `json:"public_config,omitempty"`
	PublicIPs     uint64   `json:"public_ips,omitempty"`
	HRU           uint64   `json:"hru,omitempty"` // in GB
	SRU           uint64   `json:"sru,omitempty"` // in GB
	CRU           uint64   `json:"cru,omitempty"`
	MRU           uint64   `json:"mru,omitempty"` // in GB
}

func buildNodeOptions(r *Request) NodeFilterOption {
	options := NodeFilterOption{}
	if r.Capacity.HRU != 0 {
		options.HRU = r.Capacity.HRU / (1024 * 1024 * 1024)
	}

	if r.Capacity.SRU != 0 {
		options.SRU = r.Capacity.SRU / (1024 * 1024 * 1024)
	}

	if r.Capacity.MRU != 0 {
		options.MRU = r.Capacity.MRU / (1024 * 1024 * 1024)
	}

	if r.Capacity.CRU != 0 {
		options.CRU = r.Capacity.CRU
	}

	if len(r.NodeExclude) != 0 {
		options.NodesExcluded = append(options.NodesExcluded, r.NodeExclude...)
	}

	if r.Dedicated {
		options.Dedicated = r.Dedicated
	}

	if r.PublicConfig {
		options.PublicConfig = r.PublicConfig
	}

	if r.PublicIpsCount > 0 {
		options.PublicIPs = uint64(r.PublicIpsCount)
	}

	if r.Certified {
		options.Certified = r.Certified
	}

	return options
}
