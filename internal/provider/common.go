package provider

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	gormb "github.com/threefoldtech/go-rmb"
	substrate "github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/terraform-provider-grid/internal/gridproxy"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

const RMB_WORKERS = 10

type NodeClientCollection interface {
	getNodeClient(sub *substrate.Substrate, nodeID uint32) (*client.NodeClient, error)
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

func waitDeployment(ctx context.Context, nodeClient *client.NodeClient, deploymentID uint64, version uint32, workloadVersions map[string]uint32) error {
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

func startRmbIfNeeded(ctx context.Context, api *apiClient) {
	if api.use_rmb_proxy {
		return
	}
	rmbClient, err := gormb.NewServer(api.manager, "127.0.0.1:6379", RMB_WORKERS, api.identity)
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
			data, err := wl.WorkloadData()
			if err != nil {
				log.Printf("couldn't parse workload data %s", err.Error())
				continue
			}
			if data.(*zos.PublicIP).V4 {
				res++
			}
		}
	}
	return res
}
func flistChecksumURL(url string) string {
	return fmt.Sprintf("%s.md5", url)
}
func getFlistChecksum(url string) (string, error) {
	response, err := http.Get(flistChecksumURL(url))
	if err != nil {
		return "", err
	}
	hash, err := ioutil.ReadAll(response.Body)
	return strings.TrimSpace(string(hash)), err
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
func constructWorkloadVersions(dl gridtypes.Deployment) map[string]uint32 {
	versions := make(map[string]uint32)

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

func getDeploymentObjects(ctx context.Context, sub *substrate.Substrate, dls map[uint32]uint64, nc NodeClientCollection) (map[uint32]gridtypes.Deployment, error) {
	res := make(map[uint32]gridtypes.Deployment)
	for nodeID, dlID := range dls {
		nc, err := nc.getNodeClient(sub, nodeID)
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

func isNodesUp(ctx context.Context, sub *substrate.Substrate, nodes []uint32, nc NodeClientCollection) error {
	for _, node := range nodes {
		cl, err := nc.getNodeClient(sub, node)
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

func capacity(dl gridtypes.Deployment) (gridtypes.Capacity, error) {
	cap := gridtypes.Capacity{}
	for _, wl := range dl.Workloads {
		wlCap, err := wl.Capacity()
		if err != nil {
			return cap, err
		}
		cap.Add(&wlCap)
	}
	return cap, nil
}

func capacityPrettyPrint(cap gridtypes.Capacity) string {
	return fmt.Sprintf("[mru: %d, sru: %d, hru: %d]", cap.MRU, cap.SRU, cap.HRU)
}

func hasWorkload(dl *gridtypes.Deployment, wlType gridtypes.WorkloadType) bool {
	for _, wl := range dl.Workloads {
		if wl.Type == wlType {
			return true
		}
	}
	return false
}

func ValidateDeployments(ctx context.Context, sub *substrate.Substrate, gridClient gridproxy.GridProxyClient, oldDeployments map[uint32]gridtypes.Deployment, newDeployments map[uint32]gridtypes.Deployment) error {
	farmIPs := make(map[int]int)
	nodeMap := make(map[uint32]gridproxy.NodeInfo)
	for node := range oldDeployments {
		nodeInfo, err := gridClient.Node(node)
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
		nodeInfo, err := gridClient.Node(node)
		if err != nil {
			return errors.Wrapf(err, "couldn't get node %d data from the grid proxy", node)
		}
		nodeMap[node] = nodeInfo
		farmIPs[nodeInfo.FarmID] = 0
	}
	for farm := range farmIPs {
		farmUint64 := uint64(farm)
		farmInfo, err := gridClient.Farms(gridproxy.FarmFilter{
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

// deployDeployments transforms oldDeployment to match newDeployment. In case of error,
//                   it tries to revert to the old state. Whatever is done the current state is returned
func deployDeployments(ctx context.Context, sub *substrate.Substrate, oldDeploymentIDs map[uint32]uint64, newDeployments map[uint32]gridtypes.Deployment, nc NodeClientCollection, api *apiClient, revertOnFailure bool) (map[uint32]uint64, error) {
	oldDeployments, oldErr := getDeploymentObjects(ctx, sub, oldDeploymentIDs, nc)
	if oldErr == nil {
		// check resources only when old deployments are readable
		// being readable means it's a fresh deployment or an update with good nodes
		// this is done to avoid preventing deletion of deployments on dead nodes
		if err := ValidateDeployments(ctx, sub, api.grid_client, oldDeployments, newDeployments); err != nil {
			return oldDeploymentIDs, err
		}
	}
	// ignore oldErr until we need oldDeployments
	curentDeployments, err := deployConsistentDeployments(ctx, sub, oldDeploymentIDs, newDeployments, nc, api)
	if err != nil && revertOnFailure {
		if oldErr != nil {
			return curentDeployments, fmt.Errorf("failed to deploy deployments: %w; failed to fetch deployment objects to revert deployments: %s; try again", err, oldErr)
		}

		currentDls, rerr := deployConsistentDeployments(ctx, sub, curentDeployments, oldDeployments, nc, api)
		if rerr != nil {
			return currentDls, fmt.Errorf("failed to deploy deployments: %w; failed to revert deployments: %s; try again", err, rerr)
		}
		return currentDls, err
	}
	return curentDeployments, err
}

func deployConsistentDeployments(ctx context.Context, sub *substrate.Substrate, oldDeployments map[uint32]uint64, newDeployments map[uint32]gridtypes.Deployment, nc NodeClientCollection, api *apiClient) (currentDeployments map[uint32]uint64, err error) {
	currentDeployments = make(map[uint32]uint64)
	for nodeID, contractID := range oldDeployments {
		currentDeployments[nodeID] = contractID
	}
	// deletions
	for node, contractID := range oldDeployments {
		if _, ok := newDeployments[node]; !ok {
			client, err := nc.getNodeClient(sub, node)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get node client")
			}

			err = sub.CancelContract(api.identity, contractID)

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
			client, err := nc.getNodeClient(sub, node)
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
			contractID, err := sub.CreateNodeContract(api.identity, node, nil, hashHex, publicIPCount)
			log.Printf("CreateNodeContract returned id: %d\n", contractID)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to create contract")
			}
			dl.ContractID = contractID
			ctx2, cancel := context.WithTimeout(ctx, 4*time.Minute)
			defer cancel()
			err = client.DeploymentDeploy(ctx2, dl)

			if err != nil {
				rerr := sub.CancelContract(api.identity, contractID)
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
			err = waitDeployment(ctx, client, dl.ContractID, dl.Version, newWorkloadVersions)

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

			client, err := nc.getNodeClient(sub, node)
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
			contractID, err := sub.UpdateNodeContract(api.identity, dl.ContractID, nil, hashHex)
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

			err = waitDeployment(ctx, client, dl.ContractID, dl.Version, newWorkloadsVersions)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "error waiting deployment")
			}
		}
	}

	return currentDeployments, nil
}
