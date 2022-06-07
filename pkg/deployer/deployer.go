package deployer

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/terraform-provider-grid/internal/gridproxy"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

type Deployer interface {
	Deploy(ctx context.Context, sub subi.SubstrateClient, oldDeployments map[uint32]uint64, newDeployments map[uint32]gridtypes.Deployment) (map[uint32]uint64, error)
}

type DeployerImpl struct {
	identity        substrate.Identity
	twinID          uint32
	gridClient      gridproxy.GridProxyClient
	ncPool          client.NodeClientCollection
	revertOnFailure bool
}

func NewDeployer(
	identity substrate.Identity,
	twinID uint32,
	gridClient gridproxy.GridProxyClient,
	ncPool client.NodeClientCollection,
	revertOnFailure bool,
) Deployer {
	return &DeployerImpl{
		identity,
		twinID,
		gridClient,
		ncPool,
		revertOnFailure,
	}
}

func (d *DeployerImpl) Deploy(ctx context.Context, sub subi.SubstrateClient, oldDeploymentIDs map[uint32]uint64, newDeployments map[uint32]gridtypes.Deployment) (map[uint32]uint64, error) {
	oldDeployments, oldErr := GetDeploymentObjects(ctx, sub, oldDeploymentIDs, d.ncPool)
	if oldErr == nil {
		// check resources only when old deployments are readable
		// being readable means it's a fresh deployment or an update with good nodes
		// this is done to avoid preventing deletion of deployments on dead nodes
		if err := d.Validate(ctx, sub, oldDeployments, newDeployments); err != nil {
			return oldDeploymentIDs, err
		}
	}
	// ignore oldErr until we need oldDeployments
	curentDeployments, err := d.deploy(ctx, sub, oldDeploymentIDs, newDeployments, d.revertOnFailure)
	if err != nil && d.revertOnFailure {
		if oldErr != nil {
			return curentDeployments, fmt.Errorf("failed to deploy deployments: %w; failed to fetch deployment objects to revert deployments: %s; try again", err, oldErr)
		}

		currentDls, rerr := d.deploy(ctx, sub, curentDeployments, oldDeployments, false)
		if rerr != nil {
			return currentDls, fmt.Errorf("failed to deploy deployments: %w; failed to revert deployments: %s; try again", err, rerr)
		}
		return currentDls, err
	}
	return curentDeployments, err
}

func (d *DeployerImpl) deploy(
	ctx context.Context,
	sub subi.SubstrateClient,
	oldDeployments map[uint32]uint64,
	newDeployments map[uint32]gridtypes.Deployment,
	revertOnFailure bool,
) (currentDeployments map[uint32]uint64, err error) {
	currentDeployments = make(map[uint32]uint64)
	for nodeID, contractID := range oldDeployments {
		currentDeployments[nodeID] = contractID
	}
	// deletions
	for node, contractID := range oldDeployments {
		if _, ok := newDeployments[node]; !ok {
			client, err := d.ncPool.GetNodeClient(sub, node)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get node client")
			}

			err = EnsureContractCanceled(sub, d.identity, contractID)

			if err != nil && !strings.Contains(err.Error(), "ContractNotExists") {
				return currentDeployments, errors.Wrap(err, "failed to delete deployment")
			}
			delete(currentDeployments, node)
			sub, cancel := context.WithTimeout(ctx, 1*time.Minute)
			defer cancel()
			err = client.DeploymentDelete(sub, contractID)
			if err != nil {
				log.Printf("failed to send deployment delete request to node %s", err)
			}
		}
	}
	// creations
	for node, dl := range newDeployments {
		if _, ok := oldDeployments[node]; !ok {
			client, err := d.ncPool.GetNodeClient(sub, node)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get node client")
			}

			if err := dl.Sign(d.twinID, d.identity); err != nil {
				return currentDeployments, errors.Wrap(err, "error signing deployment")
			}

			if err := dl.Valid(); err != nil {
				return currentDeployments, errors.Wrap(err, "deployment is invalid")
			}

			hash, err := dl.ChallengeHash()
			log.Printf("[DEBUG] HASH: %#v", hash)

			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to create hash")
			}

			hashHex := hex.EncodeToString(hash)

			publicIPCount := countDeploymentPublicIPs(dl)
			log.Printf("Number of public ips: %d\n", publicIPCount)
			contractID, err := sub.CreateNodeContract(d.identity, node, nil, hashHex, publicIPCount)
			log.Printf("CreateNodeContract returned id: %d\n", contractID)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to create contract")
			}
			dl.ContractID = contractID
			ctx2, cancel := context.WithTimeout(ctx, 4*time.Minute)
			defer cancel()
			err = client.DeploymentDeploy(ctx2, dl)

			if err != nil {
				rerr := EnsureContractCanceled(sub, d.identity, contractID)
				log.Printf("failed to send deployment deploy request to node %s", err)
				if rerr != nil {
					return currentDeployments, fmt.Errorf("error sending deployment to the node: %w, error cancelling contract: %s; you must cancel it manually (id: %d)", err, rerr, contractID)
				} else {
					return currentDeployments, errors.Wrap(err, "error sending deployment to the node")
				}
			}
			currentDeployments[node] = dl.ContractID
			newWorkloadVersions := map[string]uint32{}
			for _, w := range dl.Workloads {
				newWorkloadVersions[w.Name.String()] = 0
			}
			err = d.Wait(ctx, client, dl.ContractID, dl.Version, newWorkloadVersions)

			if err != nil {
				return currentDeployments, errors.Wrap(err, "error waiting deployment")
			}
		}
	}

	// updates
	for node, dl := range newDeployments {
		if oldDeploymentID, ok := oldDeployments[node]; ok {
			newDeploymentHash, err := hashDeployment(dl)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "couldn't get deployment hash")
			}

			client, err := d.ncPool.GetNodeClient(sub, node)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get node client")
			}
			oldDl, err := client.DeploymentGet(ctx, oldDeploymentID)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get old deployment to update it")
			}
			oldDeploymentHash, err := hashDeployment(oldDl)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "couldn't get deployment hash")
			}
			if oldDeploymentHash == newDeploymentHash && sameWorkloadsNames(dl, oldDl) {
				continue
			}
			oldHashes, err := constructWorkloadHashes(oldDl)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "couldn't get old workloads hashes")
			}
			newHashes, err := constructWorkloadHashes(dl)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "couldn't get new workloads hashes")
			}
			oldWorkloadsVersions := constructWorkloadVersions(oldDl)
			newWorkloadsVersions := map[string]uint32{}
			dl.Version = oldDl.Version + 1
			dl.ContractID = oldDl.ContractID
			for idx, w := range dl.Workloads {
				newHash := newHashes[string(w.Name)]
				oldHash, ok := oldHashes[string(w.Name)]
				if !ok || newHash != oldHash {
					dl.Workloads[idx].Version = dl.Version
				} else if ok && newHash == oldHash {
					dl.Workloads[idx].Version = oldWorkloadsVersions[string(w.Name)]
				}
				newWorkloadsVersions[w.Name.String()] = dl.Workloads[idx].Version
			}

			if err := dl.Sign(d.twinID, d.identity); err != nil {
				return currentDeployments, errors.Wrap(err, "error signing deployment")
			}

			if err := dl.Valid(); err != nil {
				return currentDeployments, errors.Wrap(err, "deployment is invalid")
			}

			hash, err := dl.ChallengeHash()

			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to create hash")
			}

			hashHex := hex.EncodeToString(hash)
			log.Printf("[DEBUG] HASH: %s", hashHex)
			// TODO: Destroy and create if publicIPCount is changed
			// publicIPCount := countDeploymentPublicIPs(dl)
			contractID, err := sub.UpdateNodeContract(d.identity, dl.ContractID, nil, hashHex)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to update deployment")
			}
			dl.ContractID = contractID
			sub, cancel := context.WithTimeout(ctx, 4*time.Minute)
			defer cancel()
			err = client.DeploymentUpdate(sub, dl)
			if err != nil {
				// cancel previous contract
				log.Printf("failed to send deployment update request to node %s", err)
				return currentDeployments, errors.Wrap(err, "error sending deployment to the node")
			}
			currentDeployments[node] = dl.ContractID

			err = d.Wait(ctx, client, dl.ContractID, dl.Version, newWorkloadsVersions)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "error waiting deployment")
			}
		}
	}

	return currentDeployments, nil
}

// func (d *DeployerImpl)
// Validate is a best effort validation. it returns an error if it's very sure there's a problem
//          errors that may arise because of dead nodes are ignored.
//          if a real error dodges the validation, it'll be fail anyway in the deploying phase
func (d *DeployerImpl) Validate(ctx context.Context, sub subi.SubstrateClient, oldDeployments map[uint32]gridtypes.Deployment, newDeployments map[uint32]gridtypes.Deployment) error {
	farmIPs := make(map[int]int)
	nodeMap := make(map[uint32]gridproxy.NodeInfo)
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
		farmInfo, err := d.gridClient.Farms(gridproxy.FarmFilter{
			FarmID: &farmUint64,
		}, gridproxy.Limit{
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
		farmIPs[nodeData.FarmID] += int(countDeploymentPublicIPs(dl))
	}
	for node, dl := range newDeployments {
		oldDl, alreadyExists := oldDeployments[node]
		if err := dl.Valid(); err != nil {
			return errors.Wrap(err, "invalid deployment")
		}
		needed, err := capacity(dl)
		if err != nil {
			return err
		}

		requiredIPs := int(countDeploymentPublicIPs(dl))
		nodeInfo := nodeMap[node]
		if alreadyExists {
			oldCap, err := capacity(oldDl)
			if err != nil {
				return errors.Wrapf(err, "couldn't read old deployment %d of node %d capacity", oldDl.ContractID, node)
			}
			nodeInfo.Capacity.Total.Add(&oldCap)
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
		if hasWorkload(&dl, zos.GatewayFQDNProxyType) && nodeInfo.PublicConfig.Ipv4 == "" {
			return fmt.Errorf("node %d can't deploy a fqdn workload as it doesn't have a public ipv4 configured", node)
		}
		if hasWorkload(&dl, zos.GatewayNameProxyType) && nodeInfo.PublicConfig.Domain == "" {
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
			return fmt.Errorf("node %d doesn't have enough resources. needed: %v, free: %v", node, capacityPrettyPrint(needed), capacityPrettyPrint(free))
		}
	}
	return nil
}

type Progress struct {
	time    time.Time
	stateOk int
}

func getExponentialBackoff(initial_interval time.Duration, multiplier float64, max_interval time.Duration, max_elapsed_time time.Duration) *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = initial_interval
	b.Multiplier = multiplier
	b.MaxInterval = max_interval
	b.MaxElapsedTime = max_elapsed_time
	return b
}

func (d *DeployerImpl) Wait(
	ctx context.Context,
	nodeClient *client.NodeClient,
	deploymentID uint64,
	version uint32,
	workloadVersions map[string]uint32,
) error {
	lastProgress := Progress{time.Now(), 0}
	numberOfWorkloads := len(workloadVersions)

	deploymentError := backoff.Retry(func() error {
		var deploymentVersionError error = errors.New("deployment version not updated on node")
		stateOk := 0
		sub, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		dl, err := nodeClient.DeploymentGet(sub, deploymentID)
		if err != nil {
			return backoff.Permanent(err)
		}
		if dl.Version == version {
			deploymentVersionError = nil
			for idx, wl := range dl.Workloads {
				if _, ok := workloadVersions[wl.Name.String()]; ok && wl.Version == workloadVersions[wl.Name.String()] {
					if wl.Result.State == gridtypes.StateOk {
						stateOk++
					} else if wl.Result.State == gridtypes.StateError {
						return backoff.Permanent(errors.New(fmt.Sprintf("workload %d failed within deployment %d with error %s", idx, deploymentID, wl.Result.Error)))
					}
				}
			}
		}

		if stateOk == numberOfWorkloads {
			return nil
		}

		currentProgress := Progress{time.Now(), stateOk}
		if lastProgress.stateOk < currentProgress.stateOk {
			lastProgress = currentProgress
		} else if currentProgress.time.Sub(lastProgress.time) > 4*time.Minute {
			timeoutError := fmt.Errorf("waiting for deployment %d timedout", deploymentID)
			if deploymentVersionError != nil {
				timeoutError = fmt.Errorf(timeoutError.Error()+": %w", deploymentVersionError)
			}
			return backoff.Permanent(timeoutError)
		}

		return errors.New("deployment in progress")
	},
		backoff.WithContext(getExponentialBackoff(3*time.Second, 1.25, 40*time.Second, 50*time.Minute), ctx))

	return deploymentError
}
