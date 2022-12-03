package deployer

// MultiDeployer handles resources that have multiple deployments per reservation contract
type MultiDeployer interface {
	// Create handles deployment creations
	Create()
	// Update handles deployment updates
	Update()
	// Delete handles deployment deletions
	Delete()
	// GetCurrent get current deployments
	Wait()
	// GetCurrent()
}
