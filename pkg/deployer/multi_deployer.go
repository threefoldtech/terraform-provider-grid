package deployer

import (
	"context"
)

// MultiDeployer handles resources that have multiple deployments per reservation contract
type MultiDeployerInterface interface {
	// Create handles multiple deployments creations
	Create(ctx context.Context, cl Client, data DeploymentData, d []DeployentProps) error
	// Update handles multiple deployments updates
	Update(ctx context.Context, cl Client, data DeploymentData, d []DeployentProps) error
	// Delete handles multiple deployments deletions
	Delete(ctx context.Context, cl Client, deploymentID []DeploymentID) error
}

type MultiDeployer struct {
	single SingleDeployer
}

func (m *MultiDeployer) Create(ctx context.Context, cl Client, data DeploymentData, d []DeployentProps) error {
	for _, deploymentProps := range d {
		err := m.single.Create(ctx, cl, data, deploymentProps)
		if err != nil {
			return err
		}
	}
	return nil
}
func (m *MultiDeployer) Update(ctx context.Context, cl Client, data DeploymentData, d []DeployentProps) error {
	for _, deploymentProps := range d {
		err := m.single.Update(ctx, cl, data, deploymentProps)
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
