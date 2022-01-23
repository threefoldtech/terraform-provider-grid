package gridproxy

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type GridProxyClient struct {
	endpoint string
}

func NewGridProxyClient(endpoint string) GridProxyClient {
	if endpoint[len(endpoint)-1] != '/' {
		endpoint += "/"
	}
	return GridProxyClient{endpoint}
}

func (g *GridProxyClient) url(sub string, args ...interface{}) string {
	return g.endpoint + fmt.Sprintf(sub, args...)
}

func (g *GridProxyClient) Nodes() (res []Node, err error) {
	req, err := http.Get(g.url("nodes?max_result=99999999"))
	if err != nil {
		return
	}
	if err := json.NewDecoder(req.Body).Decode(&res); err != nil {
		return res, err
	}
	return
}

func (g *GridProxyClient) AliveNodes() (res []Node, err error) {
	res, err = g.Nodes()
	n := 0
	for i := range res {
		if res[i].Status == NodeUP {
			res[n] = res[i]
			n++
		}
	}
	return
}

func (g *GridProxyClient) Farms() (res FarmResult, err error) {
	req, err := http.Get(g.url("farms?max_result=99999999"))
	if err != nil {
		return
	}
	if err := json.NewDecoder(req.Body).Decode(&res); err != nil {
		return res, err
	}
	return
}

func (g *GridProxyClient) Node(nodeID uint32) (res NodeInfo, err error) {
	req, err := http.Get(g.url("nodes/%d", nodeID))
	if err != nil {
		return
	}
	if err := json.NewDecoder(req.Body).Decode(&res); err != nil {
		return res, err
	}
	return
}

func (g *GridProxyClient) NodeStatus(nodeID uint32) (res NodeStatus, err error) {
	req, err := http.Get(g.url("nodes/%d/status", nodeID))
	if err != nil {
		return
	}
	if err := json.NewDecoder(req.Body).Decode(&res); err != nil {
		return res, err
	}
	return
}
