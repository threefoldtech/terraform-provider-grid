package gridproxy

import (
	"log"
	"time"

	"github.com/cenkalti/backoff/v3"
)

type RetryingGridProxyClient struct {
	cl      GridProxyClient
	timeout time.Duration
}

func NewRetryingGridProxyClient(cl GridProxyClient) GridProxyClient {
	return NewRetryingGridProxyClientWithTimeout(cl, 2*time.Minute)
}
func NewRetryingGridProxyClientWithTimeout(cl GridProxyClient, timeout time.Duration) GridProxyClient {
	proxy := RetryingGridProxyClient{cl, timeout}
	return &proxy
}
func bf(timeout time.Duration) *backoff.ExponentialBackOff {
	res := backoff.NewExponentialBackOff()
	res.MaxElapsedTime = timeout
	return res
}

func notify(cmd string) func(error, time.Duration) {
	return func(err error, duration time.Duration) {
		log.Printf("failure: %s, command: %s, duration: %s", err.Error(), cmd, duration)
	}
}

func (g *RetryingGridProxyClient) Ping() error {
	f := func() error {
		return g.cl.Ping()
	}
	return backoff.RetryNotify(f, bf(g.timeout), notify("ping"))

}

func (g *RetryingGridProxyClient) Nodes(filter NodeFilter, pagination Limit) (res []Node, err error) {
	f := func() error {
		res, err = g.cl.Nodes(filter, pagination)
		return err
	}
	backoff.RetryNotify(f, bf(g.timeout), notify("nodes"))
	return
}

func (g *RetryingGridProxyClient) Farms(filter FarmFilter, pagination Limit) (res FarmResult, err error) {
	f := func() error {
		res, err = g.cl.Farms(filter, pagination)
		return err
	}
	backoff.RetryNotify(f, bf(g.timeout), notify("farms"))
	return
}

func (g *RetryingGridProxyClient) Node(nodeID uint32) (res NodeInfo, err error) {
	f := func() error {
		res, err = g.cl.Node(nodeID)
		return err
	}
	backoff.RetryNotify(f, bf(g.timeout), notify("node"))
	return
}

func (g *RetryingGridProxyClient) NodeStatus(nodeID uint32) (res NodeStatus, err error) {
	f := func() error {
		res, err = g.cl.NodeStatus(nodeID)
		return err
	}
	backoff.RetryNotify(f, bf(g.timeout), notify("node_status"))
	return
}
