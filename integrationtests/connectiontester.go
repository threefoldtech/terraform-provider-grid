// Package integration includes integration  and connection testing utilities to run the tests
package integrationtests

import (
	"net"
	"time"
)

// TestConnection used to test connection
func TestConnection(addr string, port string) bool {
	connected := false
	con, err := net.DialTimeout("tcp", net.JoinHostPort(addr, port), time.Second*12)
	if err == nil {
		connected = true
	}
	con.Close()
	return connected
}
