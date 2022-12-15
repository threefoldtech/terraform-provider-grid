package deployer

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	proxy "github.com/threefoldtech/grid_proxy_server/pkg/client"
	proxytypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

type Validator interface {
	Validate(ctx context.Context, sub subi.Substrate, oldDeployments map[uint64]gridtypes.Deployment, newDeployments map[uint64]gridtypes.Deployment) error
}

type ValidatorImpl struct {
	gridClient proxy.Client
}

// Validate is a best effort validation. it returns an error if it's very sure there's a problem
//
//	errors that may arise because of dead nodes are ignored.
//	if a real error dodges the validation, it'll be fail anyway in the deploying phase
func (d *ValidatorImpl) Validate(ctx context.Context, sub subi.Substrate, oldDeployments map[uint64]gridtypes.Deployment, newDeployments map[uint64]gridtypes.Deployment) error {
	farmIPs := make(map[int]int)
	nodeMap := make(map[uint32]proxytypes.NodeWithNestedCapacity)
	for capacityID := range oldDeployments {
		contract, err := sub.GetContract(capacityID)
		if err != nil {
			return errors.Wrapf(err, "failed to get capacity contract with id %d",contract.ContractID)
		}
		nodeID := contract.ContractType.CapacityReservationContract.NodeID
		nodeInfo, err := d.gridClient.Node(uint32(nodeID))
		if err != nil {
			return errors.Wrapf(err, "couldn't get node %d data with capacity id %d from the grid proxy ", nodeID,capacityID)
		}
		nodeMap[uint32(nodeID)] = nodeInfo
		farmIPs[nodeInfo.FarmID] = 0
	}
	for capacityID := range newDeployments {
		contract, err := sub.GetContract(capacityID)
		if err != nil {
			return errors.Wrapf(err, "failed to get capacity contract for (%d)",capacityID)
		}
		nodeID := contract.ContractType.CapacityReservationContract.NodeID
		if _, ok := nodeMap[uint32(nodeID)]; ok {
			continue
		}
		nodeInfo, err := d.gridClient.Node(uint32(nodeID))
		if err != nil {
			return errors.Wrapf(err, "couldn't get node %d data from the grid proxy", capacityID)
		}
		nodeMap[uint32(nodeID)] = nodeInfo
		farmIPs[nodeInfo.FarmID] = 0
	}
	for farm := range farmIPs {
		farmUint64 := uint64(farm)
		farmInfo, _, err := d.gridClient.Farms(proxytypes.FarmFilter{
			FarmID: &farmUint64,
		}, proxytypes.Limit{
			Page: 1,
			Size: 1,
		})
		if err != nil {
			return errors.Wrapf(err, "couldn't get farm %d data from the grid proxy", farm)
		}
		if len(farmInfo) == 0 {
			return fmt.Errorf("farm %d not returned from the proxy", farm)
		}
		for _, ip := range farmInfo[0].PublicIps {
			if ip.ContractID == 0 {
				farmIPs[farm]++
			}
		}
	}
	for capacityID, dl := range oldDeployments {
		// dl := info.Deployment
		contract, err := sub.GetContract(capacityID)
		if err != nil {
			return errors.Wrapf(err, "failed to get capacity contract for %d",capacityID)
		}
		nodeID := contract.ContractType.CapacityReservationContract.NodeID
		nodeData, ok := nodeMap[uint32(nodeID)]
		if !ok {
			return fmt.Errorf("node with capcity %d not returned from the grid proxy", capacityID)
		}
		farmIPs[nodeData.FarmID] += int(countDeploymentPublicIPs(dl))
	}
	for capacityID, dl := range newDeployments {
		// dl := info.Deployment
		oldDl, ok := oldDeployments[capacityID]
		// oldDl := oldDlInfo
		contract, err := sub.GetContract(capacityID)
		if err != nil {
			return errors.Wrapf(err, "failed to get capacity contract for %d",capacityID)
		}
		nodeID := contract.ContractType.CapacityReservationContract.NodeID
		if err := dl.Valid(); err != nil {
			return errors.Wrap(err, "invalid deployment")
		}
		needed, err := capacity(dl)
		if err != nil {
			return err
		}

		requiredIPs := int(countDeploymentPublicIPs(dl))
		nodeInfo := nodeMap[uint32(nodeID)]
		if ok {
			oldCap, err := capacity(oldDl)
			if err != nil {
				return errors.Wrapf(err, "couldn't read old deployment %d of node %d capacity", oldDl.DeploymentID, capacityID)
			}
			addCapacity(&nodeInfo.Capacity.Total, &oldCap)
			deployment, err := sub.GetDeployment(oldDl.DeploymentID.U64())
			if err != nil {
				return errors.Wrapf(err, "couldn't get node contract %d", oldDl.DeploymentID)
			}
			current := int(deployment.PublicIPsCount)
			if requiredIPs > current {
				return fmt.Errorf(
					"currently, it's not possible to increase the number of reserved public ips in a deployment, node: %d, current: %d, requested: %d",
					capacityID,
					current,
					requiredIPs,
				)
			}
		}

		farmIPs[nodeInfo.FarmID] -= requiredIPs
		if farmIPs[nodeInfo.FarmID] < 0 {
			return fmt.Errorf("farm %d doesn't have enough public ips", nodeInfo.FarmID)
		}
		if hasWorkload(&dl, zos.GatewayFQDNProxyType) && nodeInfo.PublicConfig.Ipv4 == "" {
			return fmt.Errorf("capacity id %d can't deploy a fqdn workload as it doesn't have a public ipv4 configured", capacityID)
		}
		if hasWorkload(&dl, zos.GatewayNameProxyType) && nodeInfo.PublicConfig.Domain == "" {
			return fmt.Errorf("capacity id %d can't deploy a gateway name workload as it doesn't have a domain configured", capacityID)
		}
		mrus := nodeInfo.Capacity.Total.MRU - nodeInfo.Capacity.Used.MRU
		hrus := nodeInfo.Capacity.Total.HRU - nodeInfo.Capacity.Used.HRU
		srus := 2*nodeInfo.Capacity.Total.SRU - nodeInfo.Capacity.Used.SRU
		if mrus < needed.MRU ||
			srus < needed.SRU ||
			hrus < needed.HRU {
			free := gridtypes.Capacity{
				HRU: hrus,
				MRU: mrus,
				SRU: srus,
			}
			return fmt.Errorf("capacity id %d doesn't have enough resources. needed: %v, free: %v", capacityID, capacityPrettyPrint(needed), capacityPrettyPrint(free))
		}
	}
	return nil
}
