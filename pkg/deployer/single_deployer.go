package deployer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type CapacityReservationContractID uint64
type DeploymentData string
type DeploymentID uint64

// Client is used to talk to chain and nodes
type Client struct {
	identity substrate.Identity
	sub      *substrate.Substrate
	twin     uint32
	ncPool   client.NodeClientCollection
}

type DeployentProps struct {
	deployment   gridtypes.Deployment
	contractID   CapacityReservationContractID
	deploymentID DeploymentID
}

// SingleDeployerInterface handles resources that have single deployments per reservation contract
type SingleDeployerInterface interface {
	// Create handles deployment creations
	Create(ctx context.Context, cl Client, data DeploymentData, d DeployentProps) error
	// Update handles deployment updates
	Update(ctx context.Context, cl Client, data DeploymentData, d DeployentProps) error
	// Delete handles deployment deletions
	Delete(ctx context.Context, cl Client, deploymentID DeploymentID) error
	// // Wait waits until deployment is deployed on node
	// Wait(ctx context.Context, nodeClient *client.NodeClient, deploymentID DeploymentID, workloadVersions map[string]uint32) error
	// Revert should try to revert changes the deployer did. it should be used if deployer failed to deploy, and changes were to be canceled.
	// Revert(ctx context.Context, cl Client) error
	// GetCurrent gets current deployment from node
	// GetCurrent(ctx context.Context, cl Client, )
}

type SingleDeployer struct {
}

func (s *SingleDeployer) Create(ctx context.Context, cl Client, data DeploymentData, d DeployentProps) error {
	deployment := d.deployment
	capacityContract, err := cl.sub.GetContract(uint64(d.contractID))
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
	deploymentID, err := cl.sub.CreateDeployment(cl.identity, uint64(d.contractID), hashHex, string(data), cap.AsResources(), publicIPCount)
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
	err = wait(ctx, client, deploymentID, newWorkloadVersions)
	if err != nil {
		return errors.Wrap(err, "error waiting deployment")
	}
	return nil
}
func (s *SingleDeployer) Update(ctx context.Context, cl Client, data DeploymentData, d DeployentProps) error {
	capacityContract, err := cl.sub.GetContract(uint64(d.contractID))
	if err != nil {
		return err
	}
	node := capacityContract.ContractType.CapacityReservationContract.NodeID
	newDeploymentHash, err := hashDeployment(d.deployment)
	if err != nil {
		return errors.Wrap(err, "couldn't get deployment hash")
	}

	client, err := cl.ncPool.GetNodeClient(cl.sub, uint32(node))
	if err != nil {
		return errors.Wrap(err, "failed to get node client")
	}
	oldDl, err := client.DeploymentGet(ctx, uint64(d.deploymentID))
	if err != nil {
		return errors.Wrap(err, "failed to get old deployment to update it")
	}
	oldDeploymentHash, err := hashDeployment(oldDl)
	if err != nil {
		return errors.Wrap(err, "couldn't get deployment hash")
	}
	if oldDeploymentHash == newDeploymentHash && sameWorkloadsNames(d.deployment, oldDl) {
		return nil
	}
	oldHashes, err := constructWorkloadHashes(oldDl)
	if err != nil {
		return errors.Wrap(err, "couldn't get old workloads hashes")
	}
	newHashes, err := constructWorkloadHashes(d.deployment)
	if err != nil {
		return errors.Wrap(err, "couldn't get new workloads hashes")
	}
	oldWorkloadsVersions := constructWorkloadVersions(oldDl)
	newWorkloadsVersions := map[string]uint32{}
	d.deployment.Version = oldDl.Version + 1
	d.deployment.DeploymentID = oldDl.DeploymentID
	for idx, w := range d.deployment.Workloads {
		newHash := newHashes[string(w.Name)]
		oldHash, ok := oldHashes[string(w.Name)]
		if !ok || newHash != oldHash {
			d.deployment.Workloads[idx].Version = d.deployment.Version
		} else if ok && newHash == oldHash {
			d.deployment.Workloads[idx].Version = oldWorkloadsVersions[string(w.Name)]
		}
		newWorkloadsVersions[w.Name.String()] = d.deployment.Workloads[idx].Version
	}
	if err := d.deployment.Sign(cl.twin, cl.identity); err != nil {
		return errors.Wrap(err, "error signing deployment")
	}

	if err := d.deployment.Valid(); err != nil {
		return errors.Wrap(err, "deployment is invalid")
	}

	log.Printf("%+v", d.deployment)
	hash, err := d.deployment.ChallengeHash()

	if err != nil {
		return errors.Wrap(err, "failed to create hash")
	}

	hashHex := hash.Hex()
	log.Printf("[DEBUG] HASH: %s", hashHex)

	cap, err := d.deployment.Capacity()
	if err != nil {
		return errors.Wrapf(err, "couldn't get deployment capacity")
	}
	resources := cap.AsResources()
	err = cl.sub.UpdateDeployment(cl.identity, uint64(d.deployment.DeploymentID), hashHex, string(data), &resources)
	if err != nil {
		return errors.Wrap(err, "failed to update deployment")
	}
	// dl.ContractID = contractID
	sub, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	err = client.DeploymentUpdate(sub, d.deployment)
	if err != nil {
		// cancel previous contract
		log.Printf("failed to send deployment update request to node %s", err)
		return errors.Wrap(err, "error sending deployment to the node")
	}

	err = wait(ctx, client, uint64(d.deployment.DeploymentID), newWorkloadsVersions)
	if err != nil {
		return errors.Wrap(err, "error waiting deployment")
	}
	return nil
}
func (s *SingleDeployer) Delete(ctx context.Context, cl Client, deploymentID DeploymentID) error {
	err := EnsureDeploymentCanceled(cl.sub, cl.identity, uint64(deploymentID))
	if err != nil {
		return errors.Wrap(err, "failed to delete deployment")
	}
	return nil
}
