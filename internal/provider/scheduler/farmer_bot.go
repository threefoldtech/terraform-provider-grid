package scheduler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	FarmerBotVersionAction  = "farmerbot.farmmanager.version"
	FarmerBotFindNodeAction = "farmerbot.nodemanager.findnode"
	FarmerBotRMBFunction    = "execute_job"
)

type FarmerBotAction struct {
	Guid         string        `json:"guid"`
	TwinID       uint32        `json:"twinid"`
	Action       string        `json:"action"`
	Args         FarmerBotArgs `json:"args"`
	Result       FarmerBotArgs `json:"result"`
	State        string        `json:"state"`
	Start        uint64        `json:"start"`
	End          uint64        `json:"end"`
	GracePeriod  uint32        `json:"grace_period"`
	Error        string        `json:"error"`
	Timeout      uint32        `json:"timeout"`
	SourceTwinID uint32        `json:"src_twinid"`
	SourceAction string        `json:"src_action"`
	Dependencies []string      `json:"dependencies"`
}

type FarmerBotArgs struct {
	Args   []Args   `json:"args"`
	Params []Params `json:"params"`
}

type Args struct {
	RequiredHRU  *uint64  `json:"required_hru,omitempty"`
	RequiredSRU  *uint64  `json:"required_sru,omitempty"`
	RequiredCRU  *uint64  `json:"required_cru,omitempty"`
	RequiredMRU  *uint64  `json:"required_mru,omitempty"`
	NodeExclude  []uint32 `json:"node_exclude,omitempty"`
	Dedicated    *bool    `json:"dedicated,omitempty"`
	PublicConfig *bool    `json:"public_config,omitempty"`
	PublicIPs    *uint32  `json:"public_ips"`
	Certified    *bool    `json:"certified,omitempty"`
}

type Params struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

func (s *Scheduler) hasFarmerBot(ctx context.Context, farmID uint32) bool {
	_, err := s.getFarmInfo(farmID)
	if err != nil {
		return false
	}
	args := []Args{}
	params := []Params{}
	data := s.generateFarmerBotAction(farmID, args, params, FarmerBotVersionAction)
	var output FarmerBotAction
	dst := s.farms[farmID].farmerTwinID
	err = s.rmbClient.Call(ctx, dst, FarmerBotRMBFunction, data, &output)
	if err != nil {
		log.Printf("error pinging farmerbot %+v", err)
	}

	return err == nil
}

func (n *Scheduler) farmerBotSchedule(ctx context.Context, r *Request) (uint32, error) {
	info, err := n.getFarmInfo(r.FarmId)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get farm %d info", r.FarmId)
	}
	params := generateFarmerBotParams(r)
	args := generateFarmerBotArgs(r)
	data := n.generateFarmerBotAction(info.farmerTwinID, args, params, FarmerBotFindNodeAction)
	output := FarmerBotAction{}

	err = n.rmbClient.Call(ctx, info.farmerTwinID, FarmerBotRMBFunction, data, &output)
	if err != nil {
		return 0, err
	}
	if len(output.Result.Params) < 1 {
		return 0, errors.New("can not find a node to deploy on")
	}
	nodeId, err := strconv.ParseUint(output.Result.Params[0].Value.(string), 10, 32)
	if err != nil {
		return 0, err
	}
	log.Printf("got a node with id %d", nodeId)
	return uint32(nodeId), nil
}

func generateFarmerBotArgs(r *Request) []Args {
	return []Args{}
}
func generateFarmerBotParams(r *Request) []Params {
	params := []Params{}
	if r.Capacity.HRU != 0 {
		params = append(params, Params{Key: "required_hru", Value: r.Capacity.HRU})
	}

	if r.Capacity.SRU != 0 {
		params = append(params, Params{Key: "required_sru", Value: r.Capacity.SRU})
	}

	if r.Capacity.MRU != 0 {
		params = append(params, Params{Key: "required_mru", Value: r.Capacity.MRU})
	}

	if r.Capacity.CRU != 0 {
		params = append(params, Params{Key: "required_cru", Value: r.Capacity.CRU})
	}

	if len(r.NodeExclude) != 0 {
		value := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(r.NodeExclude)), ","), "")
		params = append(params, Params{Key: "node_exclude", Value: value})
	}

	if r.Dedicated {
		params = append(params, Params{Key: "dedicated", Value: r.Dedicated})
	}

	if r.PublicConfig {
		params = append(params, Params{Key: "public_config", Value: r.PublicConfig})
	}

	if r.PublicIpsCount > 0 {
		params = append(params, Params{Key: "public_ips", Value: r.PublicIpsCount})
	}

	if r.Certified {
		params = append(params, Params{Key: "certified", Value: r.Certified})
	}
	return params
}

func (s *Scheduler) generateFarmerBotAction(farmerTwinID uint32, args []Args, params []Params, action string) FarmerBotAction {
	return FarmerBotAction{
		Guid:   uuid.NewString(),
		TwinID: farmerTwinID,
		Action: action,
		Args: FarmerBotArgs{
			Args:   args,
			Params: params,
		},
		Result: FarmerBotArgs{
			Args:   []Args{},
			Params: []Params{},
		},
		State:        "init",
		Start:        uint64(time.Now().Unix()),
		End:          0,
		GracePeriod:  0,
		Error:        "",
		Timeout:      6000,
		SourceTwinID: uint32(s.twinID),
		Dependencies: []string{},
	}
}
