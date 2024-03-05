//go:build integration
// +build integration

package integrationtests

func convertGBToBytes(gb uint64) *uint64 {
	bytes := gb * 1024 * 1024 * 1024
	return &bytes
}

func convertMBToBytes(mb uint64) *uint64 {
	bytes := mb * 1024 * 1024
	return &bytes
}
