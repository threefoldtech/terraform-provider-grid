// Package integrationtests includes integration tests and connection testing utilities to run the tests
package integrationtests

import (
	"net"
	"time"
)

// TestConnection used to test connection
func TestConnection(addr string, port string) bool {
	connected := false
	for t := time.Now(); time.Since(t) < 3*time.Minute; {
		con, err := net.DialTimeout("tcp", net.JoinHostPort(addr, port), time.Second*12)
		if err == nil {
			connected = true
		}
		con.Close()
	}
	return connected
}
