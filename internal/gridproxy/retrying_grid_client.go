package gridproxy

import (
	"log"
	"time"

	"github.com/cenkalti/backoff/v3"
)

type RetryingGridProxyClient struct {
	cl GridProxyClient
}

func NewRetryingGridProxyClient(cl GridProxyClient) RetryingGridProxyClient {
	return RetryingGridProxyClient{cl}
}
func bf() *backoff.ExponentialBackOff {
	res := backoff.NewExponentialBackOff()
	res.MaxElapsedTime = 2 * time.Minute
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
	return backoff.RetryNotify(f, bf(), notify("ping"))

}

func (g *RetryingGridProxyClient) Nodes() (res []Node, err error) {
	f := func() error {
		res, err = g.cl.Nodes()
		return err
	}
	backoff.RetryNotify(f, bf(), notify("nodes"))
	return
}

func (g *RetryingGridProxyClient) AliveNodes() (res []Node, err error) {
	f := func() error {
		res, err = g.cl.Nodes()
		return err
	}
	backoff.RetryNotify(f, bf(), notify("alive_nodes"))
	return
}

func (g *RetryingGridProxyClient) Farms() (res FarmResult, err error) {
	f := func() error {
		res, err = g.cl.Farms()
		return err
	}
	backoff.RetryNotify(f, bf(), notify("farms"))
	return
}

func (g *RetryingGridProxyClient) Node(nodeID uint32) (res NodeInfo, err error) {
	f := func() error {
		res, err = g.cl.Node(nodeID)
		return err
	}
	backoff.RetryNotify(f, bf(), notify("node"))
	return
}

func (g *RetryingGridProxyClient) NodeStatus(nodeID uint32) (res NodeStatus, err error) {
	f := func() error {
		res, err = g.cl.NodeStatus(nodeID)
		return err
	}
	backoff.RetryNotify(f, bf(), notify("node_status"))
	return
}
