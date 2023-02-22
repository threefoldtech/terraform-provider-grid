// Package integrationtests includes integration tests for deploying solutions on the tf grid, and some utilities to test these solutions.
package integrationtests

import (
	"net"
	"time"
)

// TestConnection used to test connection
func TestConnection(addr string, port string) bool {
	for t := time.Now(); time.Since(t) < 3*time.Minute; {
		con, err := net.DialTimeout("tcp", net.JoinHostPort(addr, port), time.Second*12)
		if err == nil {
			con.Close()
			return true
		}
	}
	return false
}
