package deployer

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	proxytypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// MultiDeployer handles resources that have multiple deployments per reservation contract
type MultiDeployerInterface interface {
	// Create handles multiple deployments creations
	Create(ctx context.Context, cl Client, data DeploymentData, d []DeploymentProps) error
	// Update handles multiple deployments updates
	Update(ctx context.Context, cl Client, data DeploymentData, d []DeploymentProps) error
	// Delete handles multiple deployments deletions
	Delete(ctx context.Context, cl Client, deploymentID []DeploymentID) error
}

type MultiDeployer struct {
	Single SingleDeployer
}

func (m *MultiDeployer) Create(ctx context.Context, cl Client, data DeploymentData, d []DeploymentProps) error {
	for idx := range d {
		err := m.Single.validate(ctx, cl, &d[idx])
		if err != nil {
			return errors.Wrap(err, "error validating deployment")
		}
	}
	createdDeployments := []DeploymentID{}
	for idx := range d {
		err := m.Single.PushCreate(ctx, cl, data, &d[idx])
		if err != nil {
			// revertCreate: check created deployments and delete them
			revertErr := m.Delete(ctx, cl, createdDeployments)
			if revertErr != nil {
				return fmt.Errorf("failed to deploy: %w, failed to revert deployments: %w, try again.")
			}
			return err
		}
		createdDeployments = append(createdDeployments, DeploymentID(d[idx].Deployment.DeploymentID.U64()))
	}

	for idx := range d {
		err := m.Single.Wait(ctx, cl, &d[idx])
		if err != nil {
			return err
		}
	}
	return nil
}
func (m *MultiDeployer) Update(ctx context.Context, cl Client, data DeploymentData, d []DeploymentProps) error {
	for idx := range d {
		err := m.Single.validate(ctx, cl, &d[idx])
		if err != nil {
			return errors.Wrap(err, "error validating deployment")
		}
	}
	currentDeployments, err := m.getCurrentDeployments(ctx, cl, d)
	if err != nil {
		return errors.Wrap(err, "couldn't get current deployments")
	}
	for idx := range d {
		err := m.Single.PushUpdate(ctx, cl, data, &d[idx])
		if err != nil {
			// revertUpdate: check updated deployments and revert them
			m.reuseOldDeployments(currentDeployments, d)
			revertErr := m.Update(ctx, cl, data, d)
			if revertErr != nil {
				return fmt.Errorf("failed to update deployment: %w; failed to revert update: %s; try again", err, revertErr)
			}
			return errors.Wrap(err, "deployer failed to update deployments. update was reverted")
		}
	}
	for idx := range d {
		err := m.Single.Wait(ctx, cl, &d[idx])
		if err != nil {
			return err
		}
	}
	return nil
}
func (m *MultiDeployer) Delete(ctx context.Context, cl Client, deploymentID []DeploymentID) error {
	for _, id := range deploymentID {
		err := m.Single.Delete(ctx, cl, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MultiDeployer) getCurrentDeployments(ctx context.Context, cl Client, d []DeploymentProps) ([]gridtypes.Deployment, error) {
	currentDeployments := []gridtypes.Deployment{}
	for idx := range d {
		deployment, err := m.Single.getCurrentDeployment(ctx, cl, &d[idx])
		if err != nil {
			return nil, err
		}
		currentDeployments = append(currentDeployments, deployment)
	}
	return currentDeployments, nil
}

func (m *MultiDeployer) reuseOldDeployments(oldDeployments []gridtypes.Deployment, d []DeploymentProps) {
	for idx := range d {
		d[idx].Deployment = oldDeployments[idx]
	}
}

func (m *MultiDeployer) validate(ctx context.Context, cl Client, d []DeploymentProps) error {
	// get farm and node info
	farmIPs := map[uint64]int{}
	nodes := map[uint32]*proxytypes.NodeWithNestedCapacity{}
	for idx := range d {
		contract, err := cl.Sub.GetContract(uint64(d[idx].ContractID))
		if err != nil {
			return err
		}
		node := contract.ContractType.CapacityReservationContract.NodeID
		nodeInfo, err := getNodeInfo(cl, uint32(node), nodes)
		if err != nil {
			return errors.Wrapf(err, "couldn't get node %d data from the grid proxy", node)
		}

		farmUint64 := uint64(nodeInfo.FarmID)
		err = calculateFarmIPs(cl, farmUint64, farmIPs)
		if err != nil {
			return errors.Wrap(err, "couldn't get farm info")
		}

		oldCapacity := gridtypes.Capacity{}
		if d[idx].Deployment.DeploymentID != 0 {
			nodeClient, err := cl.NCPool.GetNodeClient(cl.Sub, uint32(node))
			if err != nil {
				return err
			}
			oldDeployment, err := nodeClient.DeploymentGet(ctx, d[idx].Deployment.DeploymentID.U64())
			if err != nil {
				return err
			}
			oldCapacity, err = oldDeployment.Capacity()
			if err != nil {
				return err
			}
		}
		newCapacity, err := d[idx].Deployment.Capacity()
		if err != nil {
			return err
		}
		requiredCapacity := capacityDiff(newCapacity, oldCapacity)
		freeHRU := nodes[uint32(node)].Capacity.Total.HRU - nodes[uint32(node)].Capacity.Total.HRU
		freeMRU := nodes[uint32(node)].Capacity.Total.MRU - nodes[uint32(node)].Capacity.Used.MRU
		freeSRU := nodes[uint32(node)].Capacity.Total.SRU - nodes[uint32(node)].Capacity.Used.SRU
		if requiredCapacity.HRU > freeHRU {
			return errors.Wrapf(ErrNotEnoughResources, "node %d doesn't have hru. needed: %d, free: %d", node, requiredCapacity.HRU, freeHRU)
		}
		nodes[uint32(node)].Capacity.Used.HRU += requiredCapacity.HRU

		if requiredCapacity.MRU > freeMRU {
			return errors.Wrapf(ErrNotEnoughResources, "node %d doesn't have mru. needed: %d, free: %d", node, requiredCapacity.MRU, freeMRU)
		}
		nodes[uint32(node)].Capacity.Used.MRU += requiredCapacity.MRU

		if requiredCapacity.SRU > freeSRU {
			return errors.Wrapf(ErrNotEnoughResources, "node %d doesn't have sru. needed: %d, free: %d", node, requiredCapacity.SRU, freeSRU)
		}
		nodes[uint32(node)].Capacity.Used.SRU += requiredCapacity.SRU

		if requiredCapacity.IPV4U > uint64(farmIPs[farmUint64]) {
			return errors.Wrapf(ErrNotEnoughResources, "farm %d doesn't have free public ips. needed: %d, free: %d", farmUint64, requiredCapacity.IPV4U, farmIPs)
		}
		farmIPs[farmUint64] -= int(requiredCapacity.IPV4U)

		if hasWorkload(&d[idx].Deployment, zos.GatewayFQDNProxyType) && nodeInfo.PublicConfig.Ipv4 == "" {
			return fmt.Errorf("node %d can't deploy a fqdn workload as it doesn't have a public ipv4 configured", node)
		}
		if hasWorkload(&d[idx].Deployment, zos.GatewayNameProxyType) && nodeInfo.PublicConfig.Domain == "" {
			return fmt.Errorf("node %d can't deploy a gateway name workload as it doesn't have a domain configured", node)
		}
	}
	return nil
}

func getNodeInfo(cl Client, nodeID uint32, nodes map[uint32]*proxytypes.NodeWithNestedCapacity) (*proxytypes.NodeWithNestedCapacity, error) {
	if _, ok := nodes[nodeID]; ok {
		// we already have this node's info, and consequently this farm's info
		return nodes[nodeID], nil
	}
	nodeInfo, err := cl.GridProxy.Node(nodeID)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't get node %d data from the grid proxy", nodeID)
	}
	nodes[nodeID] = &nodeInfo
	return nodes[nodeID], nil
}

func calculateFarmIPs(cl Client, farmID uint64, farmIPs map[uint64]int) error {
	if _, ok := farmIPs[farmID]; ok {
		// we already have this farm's info
		return nil
	}
	farmInfo, _, err := cl.GridProxy.Farms(proxytypes.FarmFilter{
		FarmID: &farmID,
	}, proxytypes.Limit{
		Page: 1,
		Size: 1,
	})
	if err != nil {
		return errors.Wrapf(err, "couldn't get farm %d data from the grid proxy", farmID)
	}
	if len(farmInfo) == 0 {
		return fmt.Errorf("farm %d not returned from the proxy", farmID)
	}
	for _, ip := range farmInfo[0].PublicIps {
		if ip.ContractID == 0 {
			farmIPs[farmID]++
		}
	}
	return nil
}
