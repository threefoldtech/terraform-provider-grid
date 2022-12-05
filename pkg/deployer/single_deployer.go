package deployer

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	proxy "github.com/threefoldtech/grid_proxy_server/pkg/client"
	proxytypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
	"github.com/threefoldtech/substrate-client"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

type CapacityReservationContractID uint64
type DeploymentData string
type DeploymentID uint64

var ErrNotEnoughResources = errors.New("not enough resources")

// Client is used to talk to chain and nodes
type Client struct {
	Identity  substrate.Identity
	Sub       *substrate.Substrate
	Twin      uint32
	NCPool    client.NodeClientCollection
	GridProxy proxy.Client
}

type DeploymentProps struct {
	Deployment gridtypes.Deployment
	ContractID CapacityReservationContractID
}

// SingleDeployerInterface handles resources that have single deployments per reservation contract
type SingleDeployerInterface interface {
	// Create handles deployment creations
	Create(ctx context.Context, cl Client, data DeploymentData, d *DeploymentProps) error
	// Update handles deployment updates
	Update(ctx context.Context, cl Client, data DeploymentData, d *DeploymentProps) error
	// Delete handles deployment deletions
	Delete(ctx context.Context, cl Client, deploymentID DeploymentID) error
}

type SingleDeployer struct {
}

func (s *SingleDeployer) Create(ctx context.Context, cl Client, data DeploymentData, d *DeploymentProps) error {
	err := s.validate(ctx, cl, d)
	if err != nil {
		return errors.Wrap(err, "deployer failed to validate deployment")
	}
	err = s.PushCreate(ctx, cl, data, d)
	if err != nil {
		// if contract was created, and deployment failed to deploy on node,
		// pushCreate cancels it's contract and returns an error
		return errors.Wrap(err, "deployer failed to deploy deployment")
	}
	err = s.Wait(ctx, cl, d)
	if err != nil {
		return errors.Wrap(err, "deployer failed while waiting for deployment")
	}
	return nil
}
func (s *SingleDeployer) Update(ctx context.Context, cl Client, data DeploymentData, d *DeploymentProps) error {
	err := s.validate(ctx, cl, d)
	if err != nil {
		return errors.Wrap(err, "deployer failed to validate deployment")
	}
	currentDeployment, err := s.getCurrentDeployment(ctx, cl, d)
	if err != nil {
		return errors.Wrap(err, "failed to get old deployment")
	}
	err = s.PushUpdate(ctx, cl, data, d)
	if err != nil {
		// if there is an error, revert Update
		d.Deployment = currentDeployment
		revertErr := s.PushUpdate(ctx, cl, data, d)
		if revertErr != nil {
			return fmt.Errorf("failed to update deployment: %w; failed to revert update: %s; try again", err, revertErr)
		}

		return errors.Wrap(err, "deployer failed to update deployment. update was reverted")
	}

	err = s.Wait(ctx, cl, d)
	if err != nil {
		return errors.Wrap(err, "deployer failed while waiting for deployment")
	}

	return nil
}
func (s *SingleDeployer) Delete(ctx context.Context, cl Client, deploymentID DeploymentID) error {
	err := EnsureDeploymentCanceled(cl.Sub, cl.Identity, uint64(deploymentID))
	if err != nil {
		return errors.Wrap(err, "failed to delete deployment")
	}
	return nil
}

func (s *SingleDeployer) validate(ctx context.Context, cl Client, d *DeploymentProps) error {
	contract, err := cl.Sub.GetContract(uint64(d.ContractID))
	if err != nil {
		return err
	}
	node := contract.ContractType.CapacityReservationContract.NodeID
	nodeInfo, err := cl.GridProxy.Node(uint32(node))
	if err != nil {
		return errors.Wrapf(err, "couldn't get node %d data from the grid proxy", node)
	}
	farmUint64 := uint64(nodeInfo.FarmID)
	farmInfo, _, err := cl.GridProxy.Farms(proxytypes.FarmFilter{
		FarmID: &farmUint64,
	}, proxytypes.Limit{
		Page: 1,
		Size: 1,
	})
	if err != nil {
		return errors.Wrapf(err, "couldn't get farm %d data from the grid proxy", farmUint64)
	}
	if len(farmInfo) == 0 {
		return fmt.Errorf("farm %d not returned from the proxy", farmUint64)
	}
	farmIPs := 0
	for _, ip := range farmInfo[0].PublicIps {
		if ip.ContractID == 0 {
			farmIPs++
		}
	}
	oldCapacity := gridtypes.Capacity{}
	if d.Deployment.DeploymentID != 0 {
		nodeClient, err := cl.NCPool.GetNodeClient(cl.Sub, uint32(node))
		if err != nil {
			return err
		}
		oldDeployment, err := nodeClient.DeploymentGet(ctx, d.Deployment.DeploymentID.U64())
		if err != nil {
			return err
		}
		oldCapacity, err = oldDeployment.Capacity()
		if err != nil {
			return err
		}
	}
	newCapacity, err := d.Deployment.Capacity()
	if err != nil {
		return err
	}
	requiredCapacity := capacityDiff(newCapacity, oldCapacity)
	freeHRU := nodeInfo.Capacity.Total.HRU - nodeInfo.Capacity.Used.HRU
	freeMRU := nodeInfo.Capacity.Total.MRU - nodeInfo.Capacity.Used.MRU
	freeSRU := nodeInfo.Capacity.Total.SRU - nodeInfo.Capacity.Used.SRU
	if requiredCapacity.HRU > freeHRU {
		return errors.Wrapf(ErrNotEnoughResources, "node %d doesn't have hru. needed: %d, free: %d", node, requiredCapacity.HRU, freeHRU)
	}
	if requiredCapacity.MRU > freeMRU {
		return errors.Wrapf(ErrNotEnoughResources, "node %d doesn't have mru. needed: %d, free: %d", node, requiredCapacity.MRU, freeMRU)
	}
	if requiredCapacity.SRU > freeSRU {
		return errors.Wrapf(ErrNotEnoughResources, "node %d doesn't have sru. needed: %d, free: %d", node, requiredCapacity.SRU, freeSRU)
	}
	if requiredCapacity.IPV4U > uint64(farmIPs) {
		return errors.Wrapf(ErrNotEnoughResources, "farm %d doesn't have free public ips. needed: %d, free: %d", farmUint64, requiredCapacity.IPV4U, farmIPs)
	}

	if hasWorkload(&d.Deployment, zos.GatewayFQDNProxyType) && nodeInfo.PublicConfig.Ipv4 == "" {
		return fmt.Errorf("node %d can't deploy a fqdn workload as it doesn't have a public ipv4 configured", node)
	}
	if hasWorkload(&d.Deployment, zos.GatewayNameProxyType) && nodeInfo.PublicConfig.Domain == "" {
		return fmt.Errorf("node %d can't deploy a gateway name workload as it doesn't have a domain configured", node)
	}
	return nil
}

func (s *SingleDeployer) PushCreate(ctx context.Context, cl Client, data DeploymentData, d *DeploymentProps) error {
	capacityContract, err := cl.Sub.GetContract(uint64(d.ContractID))
	if err != nil {
		return err
	}
	nodeID := capacityContract.ContractType.CapacityReservationContract.NodeID
	client, err := cl.NCPool.GetNodeClient(cl.Sub, uint32(nodeID))
	if err != nil {
		return errors.Wrap(err, "failed to get node client")
	}
	if err := d.Deployment.Sign(cl.Twin, cl.Identity); err != nil {
		return errors.Wrap(err, "error signing deployment")
	}
	if err := d.Deployment.Valid(); err != nil {
		return errors.Wrap(err, "deployment is invalid")
	}

	hash, err := d.Deployment.ChallengeHash()
	log.Printf("[DEBUG] HASH: %#v", hash)

	if err != nil {
		return errors.Wrap(err, "failed to create hash")
	}
	hashHex := hash.Hex()
	publicIPCount := countDeploymentPublicIPs(d.Deployment)
	log.Printf("Number of public ips: %d\n", publicIPCount)
	cap, err := d.Deployment.Capacity()
	if err != nil {
		return errors.Wrapf(err, "couldn't get deployment capacity")
	}
	deploymentID, err := cl.Sub.CreateDeployment(cl.Identity, uint64(d.ContractID), hashHex, string(data), cap.AsResources(), publicIPCount)
	log.Printf("createDeployment returned id: %d\n", deploymentID)
	if err != nil {
		return errors.Wrap(err, "failed to create deployment")
	}
	d.Deployment.DeploymentID = gridtypes.DeploymentID(deploymentID)
	ctx2, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	err = client.DeploymentDeploy(ctx2, d.Deployment)
	if err != nil {
		rerr := EnsureDeploymentCanceled(cl.Sub, cl.Identity, deploymentID)
		log.Printf("failed to send deployment deploy request to node %s", err)
		if rerr != nil {
			return fmt.Errorf("error sending deployment to the node: %w, error cancelling contract: %s; you must cancel it manually (id: %d)", err, rerr, deploymentID)
		} else {
			return errors.Wrap(err, "error sending deployment to the node")
		}
	}
	return nil
}

func (s *SingleDeployer) PushUpdate(ctx context.Context, cl Client, data DeploymentData, d *DeploymentProps) error {

	capacityContract, err := cl.Sub.GetContract(uint64(d.ContractID))
	if err != nil {
		return err
	}
	node := capacityContract.ContractType.CapacityReservationContract.NodeID
	newDeploymentHash, err := hashDeployment(d.Deployment)
	if err != nil {
		return errors.Wrap(err, "couldn't get deployment hash")
	}

	client, err := cl.NCPool.GetNodeClient(cl.Sub, uint32(node))
	if err != nil {
		return errors.Wrap(err, "failed to get node client")
	}
	oldDl, err := client.DeploymentGet(ctx, uint64(d.Deployment.DeploymentID))
	if err != nil {
		return errors.Wrap(err, "failed to get old deployment to update it")
	}
	oldDeploymentHash, err := hashDeployment(oldDl)
	if err != nil {
		return errors.Wrap(err, "couldn't get deployment hash")
	}
	if oldDeploymentHash == newDeploymentHash && sameWorkloadsNames(d.Deployment, oldDl) {
		return nil
	}
	oldHashes, err := constructWorkloadHashes(oldDl)
	if err != nil {
		return errors.Wrap(err, "couldn't get old workloads hashes")
	}
	newHashes, err := constructWorkloadHashes(d.Deployment)
	if err != nil {
		return errors.Wrap(err, "couldn't get new workloads hashes")
	}
	oldWorkloadsVersions := constructWorkloadVersions(oldDl)
	d.Deployment.Version = oldDl.Version + 1
	d.Deployment.DeploymentID = oldDl.DeploymentID
	for idx, w := range d.Deployment.Workloads {
		newHash := newHashes[string(w.Name)]
		oldHash, ok := oldHashes[string(w.Name)]
		if !ok || newHash != oldHash {
			d.Deployment.Workloads[idx].Version = d.Deployment.Version
		} else if ok && newHash == oldHash {
			d.Deployment.Workloads[idx].Version = oldWorkloadsVersions[string(w.Name)]
		}
	}
	if err := d.Deployment.Sign(cl.Twin, cl.Identity); err != nil {
		return errors.Wrap(err, "error signing deployment")
	}

	if err := d.Deployment.Valid(); err != nil {
		return errors.Wrap(err, "deployment is invalid")
	}

	log.Printf("%+v", d.Deployment)
	hash, err := d.Deployment.ChallengeHash()

	if err != nil {
		return errors.Wrap(err, "failed to create hash")
	}

	hashHex := hash.Hex()
	log.Printf("[DEBUG] HASH: %s", hashHex)

	cap, err := d.Deployment.Capacity()
	if err != nil {
		return errors.Wrapf(err, "couldn't get deployment capacity")
	}
	resources := cap.AsResources()
	err = cl.Sub.UpdateDeployment(cl.Identity, uint64(d.Deployment.DeploymentID), hashHex, string(data), &resources)
	if err != nil {
		return errors.Wrap(err, "failed to update deployment")
	}
	sub, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	err = client.DeploymentUpdate(sub, d.Deployment)
	if err != nil {
		// cancel previous contract
		log.Printf("failed to send deployment update request to node %s", err)
		return errors.Wrap(err, "error sending deployment to the node")
	}
	return nil
}

func (s *SingleDeployer) Wait(ctx context.Context, cl Client, d *DeploymentProps) error {
	lastProgress := Progress{time.Now(), 0}
	workloadsNumber := len(d.Deployment.Workloads)
	contract, err := cl.Sub.GetContract(uint64(d.ContractID))
	if err != nil {
		return err
	}
	nodeID := contract.ContractType.CapacityReservationContract.NodeID
	nodeClient, err := cl.NCPool.GetNodeClient(cl.Sub, uint32(nodeID))
	if err != nil {
		return err
	}
	workloadVersions := make(map[string]uint32)
	for _, wl := range d.Deployment.Workloads {
		workloadVersions[wl.Name.String()] = wl.Version
	}

	deploymentError := backoff.Retry(func() error {
		stateOk := 0
		deploymentChanges, err := nodeClient.DeploymentChanges(ctx, uint64(d.Deployment.DeploymentID))
		if err != nil {
			return backoff.Permanent(err)
		}

		for _, wl := range deploymentChanges {
			if _, ok := workloadVersions[wl.Name.String()]; ok && wl.Version == workloadVersions[wl.Name.String()] {
				var errString string = ""
				switch wl.Result.State {
				case gridtypes.StateOk:
					stateOk++
				case gridtypes.StateError:
					errString = fmt.Sprintf("workload %s within deployment %d failed with error: %s", wl.Name, d.Deployment.DeploymentID, wl.Result.Error)
				case gridtypes.StateDeleted:
					errString = fmt.Sprintf("workload %s state within deployment %d is deleted: %s", wl.Name, d.Deployment.DeploymentID, wl.Result.Error)
				case gridtypes.StatePaused:
					errString = fmt.Sprintf("workload %s state within deployment %d is paused: %s", wl.Name, d.Deployment.DeploymentID, wl.Result.Error)
				case gridtypes.StateUnChanged:
					errString = fmt.Sprintf("worklaod %s within deployment %d was not updated: %s", wl.Name, d.Deployment.DeploymentID, wl.Result.Error)
				}
				if errString != "" {
					return backoff.Permanent(errors.New(errString))
				}
			}
		}

		if stateOk == workloadsNumber {
			return nil
		}

		currentProgress := Progress{time.Now(), stateOk}
		if lastProgress.stateOk < currentProgress.stateOk {
			lastProgress = currentProgress
		} else if currentProgress.time.Sub(lastProgress.time) > 4*time.Minute {
			timeoutError := fmt.Errorf("waiting for deployment %d timedout", d.Deployment.DeploymentID)
			return backoff.Permanent(timeoutError)
		}

		return errors.New("deployment in progress")
	},
		backoff.WithContext(getExponentialBackoff(3*time.Second, 1.25, 40*time.Second, 50*time.Minute), ctx))

	return deploymentError
}

func capacityDiff(new gridtypes.Capacity, old gridtypes.Capacity) gridtypes.Capacity {
	cru := math.Max(float64(new.CRU)-float64(old.CRU), 0)
	sru := math.Max(float64(new.SRU)-float64(old.SRU), 0)
	hru := math.Max(float64(new.HRU)-float64(old.HRU), 0)
	mru := math.Max(float64(new.MRU)-float64(old.MRU), 0)
	ipv4u := math.Max(float64(new.IPV4U)-float64(old.IPV4U), 0)
	return gridtypes.Capacity{
		CRU:   uint64(cru),
		SRU:   gridtypes.Unit(sru),
		HRU:   gridtypes.Unit(hru),
		MRU:   gridtypes.Unit(mru),
		IPV4U: uint64(ipv4u),
	}
}

func (s *SingleDeployer) getCurrentDeployment(ctx context.Context, cl Client, d *DeploymentProps) (gridtypes.Deployment, error) {
	contract, err := cl.Sub.GetContract(uint64(d.ContractID))
	if err != nil {
		return gridtypes.Deployment{}, err
	}
	node := contract.ContractType.CapacityReservationContract.NodeID
	nodeClient, err := cl.NCPool.GetNodeClient(cl.Sub, uint32(node))
	if err != nil {
		return gridtypes.Deployment{}, err
	}
	oldDeployment, err := nodeClient.DeploymentGet(ctx, d.Deployment.DeploymentID.U64())
	if err != nil {
		return gridtypes.Deployment{}, err
	}
	return oldDeployment, nil
}
