package deployer

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	proxytypes "github.com/threefoldtech/grid_proxy_server/pkg/types"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func (d *DeployerImpl) GetDeploymentObjects(ctx context.Context, sub *substrate.Substrate, dls map[uint32]uint64) (map[uint32]gridtypes.Deployment, error) {
	res := make(map[uint32]gridtypes.Deployment)
	for nodeID, dlID := range dls {
		nc, err := d.ncPool.GetNodeClient(sub, nodeID)
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

// constructWorkloadHashes returns a mapping between (workloadname, node id) to the workload hash
func hashDeployment(dl gridtypes.Deployment) (string, error) {
	hashObj := md5.New()
	if err := dl.Challenge(hashObj); err != nil {
		return "", err
	}
	hash := string(hashObj.Sum(nil))
	return hash, nil
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

// constructWorkloadHashes returns a mapping between workloadname to the workload version
func constructWorkloadVersions(dl gridtypes.Deployment) map[string]uint32 {
	versions := make(map[string]uint32)

	for _, w := range dl.Workloads {
		key := string(w.Name)
		versions[key] = w.Version
	}

	return versions
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

func hasWorkload(dl *gridtypes.Deployment, wlType gridtypes.WorkloadType) bool {
	for _, wl := range dl.Workloads {
		if wl.Type == wlType {
			return true
		}
	}
	return false
}

func capacityPrettyPrint(cap gridtypes.Capacity) string {
	return fmt.Sprintf("[mru: %d, sru: %d, hru: %d]", cap.MRU, cap.SRU, cap.HRU)
}

func addCapacity(cap *proxytypes.Capacity, add *gridtypes.Capacity) {
	cap.CRU += add.CRU
	cap.MRU += add.MRU
	cap.SRU += add.SRU
	cap.HRU += add.HRU
}

func EnsureContractCanceled(sub *substrate.Substrate, identity substrate.Identity, contractID uint64) error {
	if contractID == 0 {
		return nil
	}
	if err := sub.CancelContract(identity, contractID); err != nil && err.Error() != "ContractNotExists" {
		return err
	}
	return nil
}
