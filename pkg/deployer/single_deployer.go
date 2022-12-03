package deployer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type CapacityReservationContractID uint64

// Client is used to talk to chain and nodes
type Client struct {
	identity substrate.Identity
	sub      *substrate.Substrate
	twin     uint32
	ncPool   client.NodeClientCollection
}

type DeploymentInfo struct {
	deployment     gridtypes.Deployment
	deploymentData string
}

type NewDeployment struct {
	info             DeploymentInfo
	contractID       CapacityReservationContractID
	solutionProvider *uint64
}

type UpdatedDeployment struct {
	info         DeploymentInfo
	deploymentID uint64
	contractID   CapacityReservationContractID
}

type Progress struct {
	time    time.Time
	stateOk int
}

// SingleDeployerInterface handles resources that have single deployments per reservation contract
type SingleDeployerInterface interface {
	// Create handles deployment creations
	Create(ctx context.Context, cl Client, newDeployment NewDeployment) error
	// Update handles deployment updates
	Update(ctx context.Context, cl Client, updatedDeployment UpdatedDeployment) error
	// Delete handles deployment deletions
	Delete(ctx context.Context, cl Client, deploymentID uint64) error
	// Wait waits until deployment is deployed on node
	Wait(ctx context.Context, nodeClient *client.NodeClient, deploymentID uint64, workloadVersions map[string]uint32) error
	// GetCurrent gets current deployment from node
	// GetCurrent(ctx context.Context, cl Client, )
}

type SingleDeployer struct {
}

func (s *SingleDeployer) Create(ctx context.Context, cl Client, newDeployment NewDeployment) error {
	deployment := newDeployment.info.deployment
	capacityContract, err := cl.sub.GetContract(uint64(newDeployment.contractID))
	if err != nil {
		return err
	}
	nodeID := capacityContract.ContractType.CapacityReservationContract.NodeID
	client, err := cl.ncPool.GetNodeClient(cl.sub, uint32(nodeID))
	if err != nil {
		return errors.Wrap(err, "failed to get node client")
	}
	if err := deployment.Sign(cl.twin, cl.identity); err != nil {
		return errors.Wrap(err, "error signing deployment")
	}
	if err := deployment.Valid(); err != nil {
		return errors.Wrap(err, "deployment is invalid")
	}

	hash, err := deployment.ChallengeHash()
	log.Printf("[DEBUG] HASH: %#v", hash)

	if err != nil {
		return errors.Wrap(err, "failed to create hash")
	}
	hashHex := hash.Hex()
	publicIPCount := countDeploymentPublicIPs(deployment)
	log.Printf("Number of public ips: %d\n", publicIPCount)
	cap, err := deployment.Capacity()
	if err != nil {
		return errors.Wrapf(err, "couldn't get deployment capacity")
	}
	deploymentID, err := cl.sub.CreateDeployment(cl.identity, uint64(newDeployment.contractID), hashHex, newDeployment.info.deploymentData, cap.AsResources(), publicIPCount)
	log.Printf("createDeployment returned id: %d\n", deploymentID)
	if err != nil {
		return errors.Wrap(err, "failed to create deployment")
	}
	deployment.DeploymentID = gridtypes.DeploymentID(deploymentID)
	ctx2, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	err = client.DeploymentDeploy(ctx2, deployment)

	if err != nil {
		rerr := EnsureDeploymentCanceled(cl.sub, cl.identity, deploymentID)
		log.Printf("failed to send deployment deploy request to node %s", err)
		if rerr != nil {
			return fmt.Errorf("error sending deployment to the node: %w, error cancelling contract: %s; you must cancel it manually (id: %d)", err, rerr, deploymentID)
		} else {
			return errors.Wrap(err, "error sending deployment to the node")
		}
	}
	newWorkloadVersions := map[string]uint32{}
	for _, w := range deployment.Workloads {
		newWorkloadVersions[w.Name.String()] = 0
	}
	err = s.Wait(ctx, client, deployment.DeploymentID.U64(), newWorkloadVersions)
	if err != nil {
		return errors.Wrap(err, "error waiting deployment")
	}
	return nil
}
func (s *SingleDeployer) Update(ctx context.Context, cl Client, updatedDeployment UpdatedDeployment) error {
	contractID := updatedDeployment.contractID
	deployment := updatedDeployment.info.deployment
	deploymentID := updatedDeployment.deploymentID

	capacityContract, err := cl.sub.GetContract(uint64(contractID))
	if err != nil {
		return err
	}
	node := capacityContract.ContractType.CapacityReservationContract.NodeID
	newDeploymentHash, err := hashDeployment(deployment)
	if err != nil {
		return errors.Wrap(err, "couldn't get deployment hash")
	}

	client, err := cl.ncPool.GetNodeClient(cl.sub, uint32(node))
	if err != nil {
		return errors.Wrap(err, "failed to get node client")
	}
	oldDl, err := client.DeploymentGet(ctx, deploymentID)
	if err != nil {
		return errors.Wrap(err, "failed to get old deployment to update it")
	}
	oldDeploymentHash, err := hashDeployment(oldDl)
	if err != nil {
		return errors.Wrap(err, "couldn't get deployment hash")
	}
	if oldDeploymentHash == newDeploymentHash && sameWorkloadsNames(deployment, oldDl) {
		return nil
	}
	oldHashes, err := constructWorkloadHashes(oldDl)
	if err != nil {
		return errors.Wrap(err, "couldn't get old workloads hashes")
	}
	newHashes, err := constructWorkloadHashes(deployment)
	if err != nil {
		return errors.Wrap(err, "couldn't get new workloads hashes")
	}
	oldWorkloadsVersions := constructWorkloadVersions(oldDl)
	newWorkloadsVersions := map[string]uint32{}
	deployment.Version = oldDl.Version + 1
	deployment.DeploymentID = oldDl.DeploymentID
	for idx, w := range deployment.Workloads {
		newHash := newHashes[string(w.Name)]
		oldHash, ok := oldHashes[string(w.Name)]
		if !ok || newHash != oldHash {
			deployment.Workloads[idx].Version = deployment.Version
		} else if ok && newHash == oldHash {
			deployment.Workloads[idx].Version = oldWorkloadsVersions[string(w.Name)]
		}
		newWorkloadsVersions[w.Name.String()] = deployment.Workloads[idx].Version
	}
	if err := deployment.Sign(cl.twin, cl.identity); err != nil {
		return errors.Wrap(err, "error signing deployment")
	}

	if err := deployment.Valid(); err != nil {
		return errors.Wrap(err, "deployment is invalid")
	}

	log.Printf("%+v", deployment)
	hash, err := deployment.ChallengeHash()

	if err != nil {
		return errors.Wrap(err, "failed to create hash")
	}

	hashHex := hash.Hex()
	log.Printf("[DEBUG] HASH: %s", hashHex)

	cap, err := deployment.Capacity()
	if err != nil {
		return errors.Wrapf(err, "couldn't get deployment capacity")
	}
	resources := cap.AsResources()
	err = cl.sub.UpdateDeployment(cl.identity, deployment.DeploymentID.U64(), hashHex, updatedDeployment.info.deploymentData, &resources)
	if err != nil {
		return errors.Wrap(err, "failed to update deployment")
	}
	// dl.ContractID = contractID
	sub, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	err = client.DeploymentUpdate(sub, deployment)
	if err != nil {
		// cancel previous contract
		log.Printf("failed to send deployment update request to node %s", err)
		return errors.Wrap(err, "error sending deployment to the node")
	}
	// currentDeployments[node] = dl.DeploymentID.U64()

	err = s.Wait(ctx, client, deployment.DeploymentID.U64(), newWorkloadsVersions)
	if err != nil {
		return errors.Wrap(err, "error waiting deployment")
	}
	return nil
}
func (s *SingleDeployer) Delete(ctx context.Context, cl Client, deploymentID uint64) error {
	err := EnsureDeploymentCanceled(cl.sub, cl.identity, deploymentID)
	if err != nil {
		return errors.Wrap(err, "failed to delete deployment")
	}
	return nil
}

func getExponentialBackoff(initial_interval time.Duration, multiplier float64, max_interval time.Duration, max_elapsed_time time.Duration) *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = initial_interval
	b.Multiplier = multiplier
	b.MaxInterval = max_interval
	b.MaxElapsedTime = max_elapsed_time
	return b
}

func (s *SingleDeployer) Wait(ctx context.Context, nodeClient *client.NodeClient, deploymentID uint64, workloadVersions map[string]uint32) error {
	lastProgress := Progress{time.Now(), 0}
	numberOfWorkloads := len(workloadVersions)

	deploymentError := backoff.Retry(func() error {
		stateOk := 0
		sub, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		deploymentChanges, err := nodeClient.DeploymentChanges(sub, deploymentID)
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
					errString = fmt.Sprintf("workload %s within deployment %d failed with error: %s", wl.Name, deploymentID, wl.Result.Error)
				case gridtypes.StateDeleted:
					errString = fmt.Sprintf("workload %s state within deployment %d is deleted: %s", wl.Name, deploymentID, wl.Result.Error)
				case gridtypes.StatePaused:
					errString = fmt.Sprintf("workload %s state within deployment %d is paused: %s", wl.Name, deploymentID, wl.Result.Error)
				case gridtypes.StateUnChanged:
					errString = fmt.Sprintf("worklaod %s within deployment %d was not updated: %s", wl.Name, deploymentID, wl.Result.Error)
				}
				if errString != "" {
					return backoff.Permanent(errors.New(errString))
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
			return backoff.Permanent(timeoutError)
		}

		return errors.New("deployment in progress")
	},
		backoff.WithContext(getExponentialBackoff(3*time.Second, 1.25, 40*time.Second, 50*time.Minute), ctx))

	return deploymentError
}
