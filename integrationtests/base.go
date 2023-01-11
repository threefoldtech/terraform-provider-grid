package integrationtests

import (
	"net"
	"time"
)

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
