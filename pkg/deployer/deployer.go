package deployer

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	proxy "github.com/threefoldtech/grid_proxy_server/pkg/client"
	"github.com/threefoldtech/substrate-client"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type Deployer interface {
	Deploy(ctx context.Context, sub subi.Substrate, oldDeployments map[uint64]uint64, newDeployments map[uint64]gridtypes.Deployment) (map[uint64]uint64, error)
	GetDeploymentObjects(ctx context.Context, sub subi.Substrate, dls map[uint64]uint64) (map[uint64]gridtypes.Deployment, error)
}

type DeployerImpl struct {
	identity         substrate.Identity
	twinID           uint32
	validator        Validator
	ncPool           client.NodeClientCollection
	revertOnFailure  bool
	solutionProvider *uint64
	deploymentData   string
}

func NewDeployer(
	identity substrate.Identity,
	twinID uint32,
	gridClient proxy.Client,
	ncPool client.NodeClientCollection,
	revertOnFailure bool,
	solutionProvider *uint64,
	deploymentData string,
) Deployer {
	return &DeployerImpl{
		identity,
		twinID,
		&ValidatorImpl{gridClient: gridClient},
		ncPool,
		revertOnFailure,
		solutionProvider,
		deploymentData,
	}
}

func (d *DeployerImpl) Deploy(ctx context.Context, sub subi.Substrate, oldDeploymentIDs map[uint64]uint64, newDeployments map[uint64]gridtypes.Deployment) (map[uint64]uint64, error) {
	oldDeployments, oldErr := d.GetDeploymentObjects(ctx, sub, oldDeploymentIDs)
	if oldErr == nil {
		// check resources only when old deployments are readable
		// being readable means it's a fresh deployment or an update with good nodes
		// this is done to avoid preventing deletion of deployments on dead nodes
		if err := d.validator.Validate(ctx, sub, oldDeployments, newDeployments); err != nil {
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

// TODO: inject CapacityReservationContractID in oldDeployments and newDeployments

func (d *DeployerImpl) deploy(
	ctx context.Context,
	sub subi.Substrate,
	oldDeployments map[uint64]uint64,
	newDeployments map[uint64]gridtypes.Deployment,
	revertOnFailure bool,
) (currentDeployments map[uint64]uint64, err error) {
	currentDeployments = make(map[uint64]uint64)
	for nodeID, contractID := range oldDeployments {
		currentDeployments[nodeID] = contractID
	}
	// deletions
	for capacityID, deploymentID := range oldDeployments {
		if _, ok := newDeployments[capacityID]; !ok {
			err = EnsureDeploymentCanceled(sub, d.identity, deploymentID)
			if err != nil && !strings.Contains(err.Error(), "ContractNotExists") {
				return currentDeployments, errors.Wrap(err, "failed to delete deployment")
			}
			delete(currentDeployments, capacityID)
		}
	}
	// creations
	for capacityID, dl := range newDeployments {
		if _, ok := oldDeployments[capacityID]; !ok {
			// dl := info.Deployment
			contract, err := sub.GetContract(capacityID)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get capacity contract")
			}
			nodeID := contract.ContractType.CapacityReservationContract.NodeID
			client, err := d.ncPool.GetNodeClient(sub, uint32(nodeID))
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
			hashHex := hash.Hex()

			publicIPCount := countDeploymentPublicIPs(dl)
			log.Printf("Number of public ips: %d\n", publicIPCount)
			// use sub.CreateDeployment
			cap, err := dl.Capacity()
			if err != nil {
				return currentDeployments, errors.Wrapf(err, "couldn't get deployment capacity")
			}
			deploymentID, err := sub.CreateDeployment(d.identity, capacityID, hashHex, d.deploymentData, cap.AsResources(), publicIPCount)
			log.Printf("createDeployment returned id: %d\n", deploymentID)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to create deployment")
			}
			dl.DeploymentID = gridtypes.DeploymentID(deploymentID)
			// contractID, err := sub.CreateNodeContract(d.identity, node, d.deploymentData, hashHex, publicIPCount, d.solutionProvider)
			// log.Printf("CreateNodeContract returned id: %d\n", contractID)
			// if err != nil {
			// 	return currentDeployments, errors.Wrap(err, "failed to create contract")
			// }
			// dl.ContractID = contractID
			ctx2, cancel := context.WithTimeout(ctx, 4*time.Minute)
			defer cancel()
			err = client.DeploymentDeploy(ctx2, dl)

			if err != nil {
				rerr := EnsureDeploymentCanceled(sub, d.identity, deploymentID)
				log.Printf("failed to send deployment deploy request to node %s", err)
				if rerr != nil {
					return currentDeployments, fmt.Errorf("error sending deployment to the node: %w, error cancelling contract: %s; you must cancel it manually (id: %d)", err, rerr, dl.DeploymentID)
				} else {
					return currentDeployments, errors.Wrap(err, "error sending deployment to the node")
				}
			}
			currentDeployments[capacityID] = dl.DeploymentID.U64()
			newWorkloadVersions := map[string]uint32{}
			for _, w := range dl.Workloads {
				newWorkloadVersions[w.Name.String()] = 0
			}
			err = d.Wait(ctx, client, dl.DeploymentID.U64(), newWorkloadVersions)

			if err != nil {
				return currentDeployments, errors.Wrap(err, "error waiting deployment")
			}
		}
	}

	// updates
	for capacityID, dl := range newDeployments {
		if oldDeploymentID, ok := oldDeployments[capacityID]; ok {
			// dl := info.Deployment
			newDeploymentHash, err := hashDeployment(dl)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "couldn't get deployment hash")
			}
			contract, err := sub.GetContract(capacityID)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get capacity contract")
			}
			nodeID := contract.ContractType.CapacityReservationContract.NodeID
			client, err := d.ncPool.GetNodeClient(sub, uint32(nodeID))
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
			dl.DeploymentID = oldDl.DeploymentID
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

			log.Printf("%+v", dl)
			hash, err := dl.ChallengeHash()

			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to create hash")
			}

			hashHex := hash.Hex()
			log.Printf("[DEBUG] HASH: %s", hashHex)
			// TODO: Destroy and create if publicIPCount is changed
			// publicIPCount := countDeploymentPublicIPs(dl)

			// use sub.UpdateDeployment
			cap, err := dl.Capacity()
			if err != nil {
				return currentDeployments, errors.Wrapf(err, "couldn't get deployment capacity")
			}
			resources := cap.AsResources()
			err = sub.UpdateDeployment(d.identity, dl.DeploymentID.U64(), hashHex, d.deploymentData, &resources)
			// contractID, err := sub.UpdateNodeContract(d.identity, dl.ContractID, "", hashHex)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to update deployment")
			}
			// dl.ContractID = contractID
			sub, cancel := context.WithTimeout(ctx, 4*time.Minute)
			defer cancel()
			err = client.DeploymentUpdate(sub, dl)
			if err != nil {
				// cancel previous contract
				log.Printf("failed to send deployment update request to node %s", err)
				return currentDeployments, errors.Wrap(err, "error sending deployment to the node")
			}
			currentDeployments[capacityID] = dl.DeploymentID.U64()

			err = d.Wait(ctx, client, dl.DeploymentID.U64(), newWorkloadsVersions)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "error waiting deployment")
			}
		}
	}

	return currentDeployments, nil
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
	workloadVersions map[string]uint32,
) error {
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
