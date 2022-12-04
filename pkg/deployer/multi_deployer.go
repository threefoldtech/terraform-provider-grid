package deployer

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// MultiDeployer handles resources that have multiple deployments per reservation contract
type MultiDeployerInterface interface {
	// Create handles multiple deployments creations
	Create(ctx context.Context, cl Client, data DeploymentData, d []DeploymentProps) error
	// Update handles multiple deployments updates
	Update(ctx context.Context, cl Client, data DeploymentData, d []DeploymentProps) error
	// Delete handles multiple deployments deletions
	Delete(ctx context.Context, cl Client, deploymentID []DeploymentID) error
}

type MultiDeployer struct {
	single SingleDeployer
}

func (m *MultiDeployer) Create(ctx context.Context, cl Client, data DeploymentData, d []DeploymentProps) error {
	for idx := range d {
		err := m.single.validate(ctx, cl, &d[idx])
		if err != nil {
			return errors.Wrap(err, "error validating deployment")
		}
	}
	createdDeployments := []DeploymentID{}
	for idx := range d {
		err := m.single.PushCreate(ctx, cl, data, &d[idx])
		if err != nil {
			// revertCreate: check created deployments and delete them
			revertErr := m.Delete(ctx, cl, createdDeployments)
			if revertErr != nil {
				return fmt.Errorf("failed to deploy: %w, failed to revert deployments: %w, try again.")
			}
			return err
		}
		createdDeployments = append(createdDeployments, DeploymentID(d[idx].deployment.DeploymentID.U64()))
	}

	for idx := range d {
		err := m.single.Wait(ctx, cl, &d[idx])
		if err != nil {
			return err
		}
	}
	return nil
}
func (m *MultiDeployer) Update(ctx context.Context, cl Client, data DeploymentData, d []DeploymentProps) error {
	for idx := range d {
		err := m.single.validate(ctx, cl, &d[idx])
		if err != nil {
			return errors.Wrap(err, "error validating deployment")
		}
	}
	currentDeployments, err := m.getCurrentDeployments(ctx, cl, d)
	if err != nil {
		return errors.Wrap(err, "couldn't get current deployments")
	}
	for idx := range d {
		err := m.single.PushUpdate(ctx, cl, data, &d[idx])
		if err != nil {
			// revertUpdate: check updated deployments and revert them
			m.reuseOldDeployments(currentDeployments, d)
			revertErr := m.Update(ctx, cl, data, d)
			if revertErr != nil {
				return fmt.Errorf("failed to update deployment: %w; failed to revert update: %s; try again", err, revertErr)
			}
			return errors.Wrap(err, "deployer failed to update deployments. update was reverted")
		}
	}
	for idx := range d {
		err := m.single.Wait(ctx, cl, &d[idx])
		if err != nil {
			return err
		}
	}
	return nil
}
func (m *MultiDeployer) Delete(ctx context.Context, cl Client, deploymentID []DeploymentID) error {
	for _, id := range deploymentID {
		err := m.single.Delete(ctx, cl, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MultiDeployer) getCurrentDeployments(ctx context.Context, cl Client, d []DeploymentProps) ([]gridtypes.Deployment, error) {
	currentDeployments := []gridtypes.Deployment{}
	for idx := range d {
		deployment, err := m.single.getCurrentDeployment(ctx, cl, &d[idx])
		if err != nil {
			return nil, err
		}
		currentDeployments = append(currentDeployments, deployment)
	}
	return currentDeployments, nil
}

func (m *MultiDeployer) reuseOldDeployments(oldDeployments []gridtypes.Deployment, d []DeploymentProps) {
	for idx := range d {
		d[idx].deployment = oldDeployments[idx]
	}
}

// implement validation
// implement revert
