package provider

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	gormb "github.com/threefoldtech/rmb"
	"github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/substrate"
)

type NodeClientCollection interface {
	getNodeClient(nodeID uint32) (*client.NodeClient, error)
}

func waitDeployment(ctx context.Context, nodeClient *client.NodeClient, deploymentID uint64, version int) error {
	done := false
	for start := time.Now(); time.Since(start) < 4*time.Minute; time.Sleep(1 * time.Second) {
		done = true
		sub, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		dl, err := nodeClient.DeploymentGet(sub, deploymentID)
		if err != nil {
			return err
		}
		if dl.Version != version {
			continue
		}
		for idx, wl := range dl.Workloads {
			if wl.Result.State == "" {
				done = false
				continue
			}
			if wl.Result.State != gridtypes.StateOk {
				return errors.New(fmt.Sprintf("workload %d failed within deployment %d with error %s", idx, deploymentID, wl.Result.Error))
			}
		}
		if done {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("waiting for deployment %d timedout", deploymentID))
}

func cancelDeployment(ctx context.Context, nc *client.NodeClient, sc *substrate.Substrate, identity substrate.Identity, id uint64) error {
	err := sc.CancelContract(&identity, id)
	if err != nil {
		return errors.Wrap(err, "error cancelling contract")
	}
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	if err := nc.DeploymentDelete(ctx, id); err != nil {
		return errors.Wrap(err, "error deleting deployment")
	}
	return nil
}

func startRmb(ctx context.Context, substrateURL string, twinID int) {
	rmbClient, err := gormb.NewServer(substrateURL, "127.0.0.1:6379", twinID)
	if err != nil {
		log.Fatalf("couldn't start server %s\n", err)
	}
	if err := rmbClient.Serve(ctx); err != nil {
		log.Printf("error serving rmb %s\n", err)
	}
}

func countDeploymentPublicIPs(dl gridtypes.Deployment) uint32 {
	var res uint32 = 0
	for _, wl := range dl.Workloads {
		if wl.Type == zos.PublicIPType {
			res++
		}
	}
	return res
}

// constructWorkloadHashes returns a mapping between (workloadname, node id) to the workload hash
func constructWorkloadHashes(deployments map[uint32]gridtypes.Deployment) (map[string]string, error) {
	hashes := make(map[string]string)

	for node, dl := range deployments {
		for _, w := range dl.Workloads {
			key := fmt.Sprintf("%d-%s", node, w.Name)
			hashObj := md5.New()
			if err := w.Challenge(hashObj); err != nil {
				return nil, errors.Wrap(err, "couldn't get new workload hash")
			}
			hash := string(hashObj.Sum(nil))
			hashes[key] = hash
		}
	}

	return hashes, nil
}

// constructWorkloadHashes returns a mapping between (workloadname, node id) to the workload version
func constructWorkloadVersions(deployments map[uint32]gridtypes.Deployment) map[string]int {
	versions := make(map[string]int)

	for node, dl := range deployments {
		for _, w := range dl.Workloads {
			key := fmt.Sprintf("%d-%s", node, w.Name)
			versions[key] = w.Version
		}
	}

	return versions
}

// constructWorkloadHashes returns a mapping between (workloadname, node id) to the workload hash
func hashDeployment(dl gridtypes.Deployment) (string, error) {
	hashObj := md5.New()
	if err := dl.Challenge(hashObj); err != nil {
		return "", err
	}
	hash := string(hashObj.Sum(nil))
	return hash, nil
}

func getDeploymentObjects(ctx context.Context, dls map[uint32]uint64, nc NodeClientCollection) (map[uint32]gridtypes.Deployment, error) {
	res := make(map[uint32]gridtypes.Deployment)
	for nodeID, dlID := range dls {
		nc, err := nc.getNodeClient(nodeID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get node client")
		}
		sub, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		dl, err := nc.DeploymentGet(sub, dlID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get deployment")
		}
		res[nodeID] = dl
	}
	return res, nil
}

func isNodeUp(ctx context.Context, nc *client.NodeClient) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := nc.NetworkListInterfaces(ctx)
	if err != nil {
		return errors.New("couldn't list node interfaces")
	}

	return nil
}

func isNodesUp(ctx context.Context, nodes []uint32, nc NodeClientCollection) error {
	for _, node := range nodes {
		cl, err := nc.getNodeClient(node)
		if err != nil {
			return fmt.Errorf("couldn't get node %d client: %w", node, err)
		}
		if err := isNodeUp(ctx, cl); err != nil {
			return fmt.Errorf("couldn't reach node %d: %w", node, err)
		}
	}

	return nil
}

func sameWorkloadsNames(d1 gridtypes.Deployment, d2 gridtypes.Deployment) bool {
	if len(d1.Workloads) != len(d2.Workloads) {
		return false
	}

	names := make(map[string]bool)
	for _, w := range d1.Workloads {
		names[string(w.Name)] = true
	}

	for _, w := range d2.Workloads {
		if _, ok := names[string(w.Name)]; !ok {
			return false
		}
	}
	return true
}

// deployDeployments transforms oldDeployment to match newDeployment. In case of error,
//                   it tries to revert to the old state. Whatever is done the current state is returned
func deployDeployments(ctx context.Context, oldDeployments map[uint32]gridtypes.Deployment, newDeployments map[uint32]gridtypes.Deployment, nc NodeClientCollection, api *apiClient, revertOnFailure bool) (map[uint32]uint64, error) {
	curentDeployments, err := deployConsistentDeployments(ctx, oldDeployments, newDeployments, nc, api)
	if err != nil && revertOnFailure {
		currentDeploymentObjects, rerr := getDeploymentObjects(ctx, curentDeployments, nc)
		if rerr != nil {
			return curentDeployments, fmt.Errorf("failed to deploy deployments: %w; failed to fetch deployment objects to revert deployments: %s; terraform apply to try again", err, rerr)
		}
		currentDls, rerr := deployConsistentDeployments(ctx, currentDeploymentObjects, oldDeployments, nc, api)
		if rerr != nil {
			return currentDls, fmt.Errorf("failed to deploy deployments: %w; failed to revert deployments: %s; terraform apply to try again", err, rerr)
		}
		return currentDls, err
	}
	return curentDeployments, err
}

func deployConsistentDeployments(ctx context.Context, oldDeployments map[uint32]gridtypes.Deployment, newDeployments map[uint32]gridtypes.Deployment, nc NodeClientCollection, api *apiClient) (currentDeployments map[uint32]uint64, err error) {

	currentDeployments = make(map[uint32]uint64)
	for nodeID, dl := range oldDeployments {
		currentDeployments[nodeID] = dl.ContractID
	}
	oldHashes, err := constructWorkloadHashes(oldDeployments)
	if err != nil {
		return currentDeployments, errors.Wrap(err, "couldn't calculate old workloads hashes")
	}
	newHashes, err := constructWorkloadHashes(newDeployments)
	if err != nil {
		return currentDeployments, errors.Wrap(err, "couldn't calculate new workloads hashes")
	}
	oldWorkloadsVersions := constructWorkloadVersions(oldDeployments)

	// deletions
	for node, dl := range oldDeployments {
		if _, ok := newDeployments[node]; !ok {
			client, err := nc.getNodeClient(node)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get node client")
			}

			err = api.sub.CancelContract(api.identity, dl.ContractID)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to delete deployment")
			}
			delete(currentDeployments, node)
			sub, cancel := context.WithTimeout(ctx, 1*time.Minute)
			defer cancel()
			err = client.DeploymentDelete(sub, dl.ContractID)
			if err != nil {
				log.Printf("failed to send deployment delete request to node %s", err)
			}
		}
	}
	// creations
	for node, dl := range newDeployments {
		if _, ok := oldDeployments[node]; !ok {
			client, err := nc.getNodeClient(node)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get node client")
			}
			if err := dl.Sign(api.twin_id, api.userSK); err != nil {
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
			contractID, err := api.sub.CreateNodeContract(api.identity, node, nil, hashHex, publicIPCount)
			log.Printf("CreateNodeContract returned id: %d\n", contractID)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to create contract")
			}
			dl.ContractID = contractID
			sub, cancel := context.WithTimeout(ctx, 4*time.Minute)
			defer cancel()
			err = client.DeploymentDeploy(sub, dl)

			if err != nil {
				rerr := api.sub.CancelContract(api.identity, contractID)
				log.Printf("failed to send deployment deploy request to node %s", err)
				if rerr != nil {
					return currentDeployments, fmt.Errorf("error sending deployment to the node: %w, error cancelling contract: %s; you must cancel it manually (id: %d)", err, rerr, contractID)
				} else {
					return currentDeployments, errors.Wrap(err, "error sending deployment to the node")
				}
			}
			currentDeployments[node] = dl.ContractID

			err = waitDeployment(ctx, client, dl.ContractID, dl.Version)

			if err != nil {
				return currentDeployments, errors.Wrap(err, "error waiting deployment")
			}
		}
	}

	// updates
	for node, dl := range newDeployments {
		newDeploymentHash, err := hashDeployment(dl)
		if err != nil {
			return currentDeployments, errors.Wrap(err, "couldn't get deployment hash")
		}
		if oldDl, ok := oldDeployments[node]; ok {
			oldDeploymentHash, err := hashDeployment(oldDl)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "couldn't get deployment hash")
			}
			if oldDeploymentHash == newDeploymentHash && sameWorkloadsNames(dl, oldDl) {
				continue
			}
			dl.Version = oldDl.Version + 1
			dl.ContractID = oldDl.ContractID
			for idx, w := range dl.Workloads {
				key := fmt.Sprintf("%d-%s", node, w.Name)
				newHash := newHashes[key]
				oldHash, ok := oldHashes[key]
				if !ok || newHash != oldHash {
					dl.Workloads[idx].Version = dl.Version
				} else if ok && newHash == oldHash {
					dl.Workloads[idx].Version = oldWorkloadsVersions[key]
				}
			}
			client, err := nc.getNodeClient(node)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get node client")
			}

			if err := dl.Sign(api.twin_id, api.userSK); err != nil {
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
			contractID, err := api.sub.UpdateNodeContract(api.identity, dl.ContractID, nil, hashHex)
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

			err = waitDeployment(ctx, client, dl.ContractID, dl.Version)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "error waiting deployment")
			}
		}
	}

	return currentDeployments, nil
}
