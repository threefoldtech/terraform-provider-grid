package gridproxy

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
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

func (g *GridProxyClient) Ping() error {
	req, err := http.Get(g.url(""))
	if err != nil {
		return err
	}
	if req.StatusCode != http.StatusOK {
		return fmt.Errorf("non ok return status code from the the grid proxy home page: %d", req.StatusCode)
	}
	return nil
}

func (g *GridProxyClient) Nodes() (res []Node, err error) {
	req, err := http.Get(g.url("nodes?size=99999999&max_result=99999999"))
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
	req, err := http.Get(g.url("farms?size=99999999&max_result=99999999"))
	if err != nil {
		return
	}
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &res)
	if err != nil || len(res) == 0 {
		var old FarmResultV0
		err1 := json.Unmarshal(data, &old)
		if err1 != nil {
			log.Printf("error unmarshaling old %s", err1.Error())
			return
		}
		res = old.Data.Farms
		err = nil
		return
	}
	return
}

func (g *GridProxyClient) Node(nodeID uint32) (res NodeInfo, err error) {
	req, err := http.Get(g.url("nodes/%d", nodeID))
	if err != nil {
		return
	}
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}
	if strings.Contains(string(data), `"total_resources"`) {
		err = json.Unmarshal(data, &res)
	} else {
		var old NodeInfoV0
		err = json.Unmarshal(data, &old)
		res = NodeInfo{
			Capacity: capacityResult{
				Used:  old.Capacity.Used,
				Total: old.Capacity.Total,
			},
			DMI:        old.DMI,
			Hypervisor: old.Hypervisor,
		}
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
