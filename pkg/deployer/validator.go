package deployer

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	proxy "github.com/threefoldtech/grid_proxy_server/pkg/client"
	proxyTypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// Validator interface for any validator
type Validator interface {
	Validate(ctx context.Context, sub subi.SubstrateExt, oldDeployments map[uint32]gridtypes.Deployment, newDeployments map[uint32]gridtypes.Deployment) error
}

// ValidatorImpl struct
type ValidatorImpl struct {
	gridClient proxy.Client
}

// Validate is a best effort validation. it returns an error if it's very sure there's a problem
//
//	errors that may arise because of dead nodes are ignored.
//	if a real error dodges the validation, it'll be fail anyway in the deploying phase
func (d *ValidatorImpl) Validate(ctx context.Context, sub subi.SubstrateExt, oldDeployments map[uint32]gridtypes.Deployment, newDeployments map[uint32]gridtypes.Deployment) error {
	farmIPs := make(map[int]int)
	nodeMap := make(map[uint32]proxyTypes.NodeWithNestedCapacity)
	for node := range oldDeployments {
		nodeInfo, err := d.gridClient.Node(node)
		if err != nil {
			return errors.Wrapf(err, "couldn't get node %d data from the grid proxy", node)
		}
		nodeMap[node] = nodeInfo
		farmIPs[nodeInfo.FarmID] = 0
	}
	for node := range newDeployments {
		if _, ok := nodeMap[node]; ok {
			continue
		}
		nodeInfo, err := d.gridClient.Node(node)
		if err != nil {
			return errors.Wrapf(err, "couldn't get node %d data from the grid proxy", node)
		}
		nodeMap[node] = nodeInfo
		farmIPs[nodeInfo.FarmID] = 0
	}
	for farm := range farmIPs {
		farmUint64 := uint64(farm)
		farmInfo, _, err := d.gridClient.Farms(proxyTypes.FarmFilter{
			FarmID: &farmUint64,
		}, proxyTypes.Limit{
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
	for node, dl := range oldDeployments {
		nodeData, ok := nodeMap[node]
		if !ok {
			return fmt.Errorf("node %d not returned from the grid proxy", node)
		}

		publicIPCount, err := CountDeploymentPublicIPs(dl)
		if err != nil {
			return errors.Wrap(err, "failed to count deployment public IPs")
		}

		farmIPs[nodeData.FarmID] += int(publicIPCount)
	}
	for node, dl := range newDeployments {
		oldDl, alreadyExists := oldDeployments[node]
		if err := dl.Valid(); err != nil {
			return errors.Wrap(err, "invalid deployment")
		}
		needed, err := Capacity(dl)
		if err != nil {
			return err
		}

		publicIPCount, err := CountDeploymentPublicIPs(dl)
		if err != nil {
			return errors.Wrap(err, "failed to count deployment public IPs")
		}
		requiredIPs := int(publicIPCount)
		nodeInfo := nodeMap[node]
		if alreadyExists {
			oldCap, err := Capacity(oldDl)
			if err != nil {
				return errors.Wrapf(err, "couldn't read old deployment %d of node %d capacity", oldDl.ContractID, node)
			}
			AddCapacity(&nodeInfo.Capacity.Total, &oldCap)
			contract, err := sub.GetContract(oldDl.ContractID)
			if err != nil {
				return errors.Wrapf(err, "couldn't get node contract %d", oldDl.ContractID)
			}
			current := int(contract.ContractType.NodeContract.PublicIPsCount)
			if requiredIPs > current {
				return fmt.Errorf(
					"currently, it's not possible to increase the number of reserved public ips in a deployment, node: %d, current: %d, requested: %d",
					node,
					current,
					requiredIPs,
				)
			}
		}

		farmIPs[nodeInfo.FarmID] -= requiredIPs
		if farmIPs[nodeInfo.FarmID] < 0 {
			return fmt.Errorf("farm %d doesn't have enough public ips", nodeInfo.FarmID)
		}
		if HasWorkload(&dl, zos.GatewayFQDNProxyType) && nodeInfo.PublicConfig.Ipv4 == "" {
			return fmt.Errorf("node %d can't deploy a fqdn workload as it doesn't have a public ipv4 configured", node)
		}
		if HasWorkload(&dl, zos.GatewayNameProxyType) && nodeInfo.PublicConfig.Domain == "" {
			return fmt.Errorf("node %d can't deploy a gateway name workload as it doesn't have a domain configured", node)
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
			return fmt.Errorf("node %d doesn't have enough resources. needed: %v, free: %v", node, CapacityPrettyPrint(needed), CapacityPrettyPrint(free))
		}
	}
	return nil
}
