package provider

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/pkg/errors"
	gormb "github.com/threefoldtech/go-rmb"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/terraform-provider-grid/internal"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

const RMB_WORKERS = 10

type NodeClientCollection interface {
	getNodeClient(nodeID uint32) (*client.NodeClient, error)
}

func diagsFromErr(e error) diag.Diagnostics {
	if e, ok := e.(*internal.Terror); ok {
		return e.Diagnostics
	}
	return diag.FromErr(e)
}

func checkWorkloadState(ctx context.Context, nodeClient *client.NodeClient, deploymentID uint64, version int) (bool, error) {
	t := internal.NewTerror()
	sub, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	dl, err := nodeClient.DeploymentGet(sub, deploymentID)
	if err != nil {
		t.AppendErr(err)
		return false, t.AsError()
	}
	if dl.Version != version {
		t.AppendErr(fmt.Errorf("version is %d, expected: %d", dl.Version, version))
		return false, t.AsError()
	}
	done := true
	for _, wl := range dl.Workloads {
		if wl.Result.State == "" {
			t.AppendErr(fmt.Errorf("state of workload %s is not a final state", wl.Name))
			done = false
			continue
		}
		if wl.Result.State != gridtypes.StateOk {
			t.Append(diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("workload %s failed", wl.Name),
				Detail:   wl.Result.Error,
			})
		}
	}
	return done, t.AsError()
}

func waitDeployment(ctx context.Context, nodeClient *client.NodeClient, deploymentID uint64, version int) error {
	t := internal.NewTerror()
	tc := time.NewTicker(1 * time.Second)
	var lerr error
	ctx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			if lerr != nil {
				t.AppendWrappedErr(lerr, "deployment %d timed out", deploymentID)
			} else {
				t.AppendErr(fmt.Errorf("deployment %d timed out", deploymentID))
			}
			return t.AsError()
		case <-tc.C:
			done, err := checkWorkloadState(ctx, nodeClient, deploymentID, version)
			log.Printf("checking round %t: %s", done, err)
			if done {
				if err != nil {
					t.AppendWrappedErr(err, "deployment %d error", deploymentID)
				}
				return t.AsError()
			}
			lerr = err
		}
	}
}

func startRmbIfNeeded(ctx context.Context, api *apiClient) {
	if api.use_rmb_proxy {
		return
	}
	rmbClient, err := gormb.NewServer(api.substrate_url, "127.0.0.1:6379", int(api.twin_id), RMB_WORKERS)
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

// constructWorkloadHashes returns a mapping between workloadname to the workload hash
func constructWorkloadHashes(dl gridtypes.Deployment) (map[string]string, error) {
	hashes := make(map[string]string)

	for _, w := range dl.Workloads {
		key := string(w.Name)
		hashObj := md5.New()
		if err := w.Challenge(hashObj); err != nil {
			return nil, errors.Wrap(err, "couldn't get new workload hash")
		}
		hash := string(hashObj.Sum(nil))
		hashes[key] = hash
	}

	return hashes, nil
}

// constructWorkloadHashes returns a mapping between workloadname to the workload version
func constructWorkloadVersions(dl gridtypes.Deployment) map[string]int {
	versions := make(map[string]int)

	for _, w := range dl.Workloads {
		key := string(w.Name)
		versions[key] = w.Version
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
			return nil, errors.Wrapf(err, "failed to get node %d client", nodeID)
		}
		sub, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		dl, err := nc.DeploymentGet(sub, dlID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get deployment %d of node %d", dlID, nodeID)
		}
		res[nodeID] = dl
	}
	return res, nil
}

func isNodeUp(ctx context.Context, nc *client.NodeClient) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := nc.NetworkListInterfaces(ctx)
	if err != nil {
		return err
	}

	return nil
}

func isNodesUp(ctx context.Context, nodes []uint32, nc NodeClientCollection) error {
	t := internal.NewTerror()
	for _, node := range nodes {
		cl, err := nc.getNodeClient(node)
		if errors.Is(err, substrate.ErrNotFound) {
			t.AppendErr(fmt.Errorf("node %d doesn't exit. make sure you are on the right network (e.g. dev/test) and the node id is correct", node))
			continue
		} else if err != nil {
			t.Append(diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("couldn't get node %d client", node),
				Detail:   err.Error(),
			})
			continue
		}
		if err := isNodeUp(ctx, cl); err != nil {
			t.Append(diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("couldn't connect to node %d, node can be down or unreachable", node),
				Detail:   err.Error(),
			})
			continue
		}
	}
	return t.AsError()
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
func deployDeployments(ctx context.Context, oldDeploymentIDs map[uint32]uint64, newDeployments map[uint32]gridtypes.Deployment, nc NodeClientCollection, api *apiClient, revertOnFailure bool) (map[uint32]uint64, error) {
	t := internal.NewTerror()
	oldDeployments, oldErr := getDeploymentObjects(ctx, oldDeploymentIDs, nc)
	// ignore oldErr until we need oldDeployments
	curentDeployments, err := deployConsistentDeployments(ctx, oldDeploymentIDs, newDeployments, nc, api)
	if err != nil {
		t.AppendWrappedErr(err, "failed to deploy deployments")
	}
	if err != nil && revertOnFailure {
		if oldErr != nil {
			t.AppendWrappedErr(oldErr, "failed to fetch deployment objects to revert deployments")
			return curentDeployments, t.AsError()
		}

		currentDls, rerr := deployConsistentDeployments(ctx, curentDeployments, oldDeployments, nc, api)
		if rerr != nil {
			t.AppendWrappedErr(rerr, "failed to revert deployments")
			return currentDls, t.AsError()
		}
		return currentDls, t.AsError()
	}
	return curentDeployments, t.AsError()
}

func deployConsistentDeployments(ctx context.Context, oldDeployments map[uint32]uint64, newDeployments map[uint32]gridtypes.Deployment, nc NodeClientCollection, api *apiClient) (currentDeployments map[uint32]uint64, err error) {

	currentDeployments = make(map[uint32]uint64)
	for nodeID, contractID := range oldDeployments {
		currentDeployments[nodeID] = contractID
	}
	// deletions
	for node, contractID := range oldDeployments {
		if _, ok := newDeployments[node]; !ok {
			client, err := nc.getNodeClient(node)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get node client")
			}

			err = api.sub.CancelContract(api.identity, contractID)
			if err != nil {
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
			client, err := nc.getNodeClient(node)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get node client")
			}

			if err := dl.Sign(api.twin_id, api.identity); err != nil {
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
				return currentDeployments, err
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

			client, err := nc.getNodeClient(node)
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
			}

			if err := dl.Sign(api.twin_id, api.identity); err != nil {
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
				return currentDeployments, err
			}
		}
	}

	return currentDeployments, nil
}
