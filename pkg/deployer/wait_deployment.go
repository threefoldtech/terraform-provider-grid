package deployer

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

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

func wait(ctx context.Context, nodeClient *client.NodeClient, deploymentID uint64, workloadVersions map[string]uint32) error {
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
