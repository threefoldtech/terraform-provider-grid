package gridproxy

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

type GridProxyClient interface {
	Ping() error
	Nodes(filter NodeFilter, pagination Limit) (res []Node, err error)
	Farms(filter FarmFilter, pagination Limit) (res FarmResult, err error)
	Node(nodeID uint32) (res NodeInfo, err error)
	NodeStatus(nodeID uint32) (res NodeStatus, err error)
}

type GridProxyClientimpl struct {
	endpoint string
}

func NewGridProxyClient(endpoint string) GridProxyClient {
	if endpoint[len(endpoint)-1] != '/' {
		endpoint += "/"
	}
	proxy := GridProxyClientimpl{endpoint}
	return &proxy
}

func parseError(body io.ReadCloser) error {
	text, err := ioutil.ReadAll(body)
	if err != nil {
		return errors.Wrap(err, "couldn't read body response")
	}
	var res ErrorReply
	if err := json.Unmarshal(text, &res); err != nil {
		return errors.New(string(text))
	}
	return fmt.Errorf("%s", res.Error)
}

func (g *GridProxyClientimpl) url(sub string, args ...interface{}) string {
	return g.endpoint + fmt.Sprintf(sub, args...)
}

func (g *GridProxyClientimpl) Ping() error {
	req, err := http.Get(g.url(""))
	if err != nil {
		return err
	}
	if req.StatusCode != http.StatusOK {
		return fmt.Errorf("non ok return status code from the the grid proxy home page: %s", http.StatusText(req.StatusCode))
	}
	return nil
}

func (g *GridProxyClientimpl) Nodes(filter NodeFilter, limit Limit) (res []Node, err error) {
	query := nodeParams(filter, limit)
	req, err := http.Get(g.url(fmt.Sprintf("nodes%s", query)))
	if err != nil {
		return
	}
	if req.StatusCode != http.StatusOK {
		err = parseError(req.Body)
		return
	}
	if err := json.NewDecoder(req.Body).Decode(&res); err != nil {
		return res, err
	}
	return
}

func (g *GridProxyClientimpl) Farms(filter FarmFilter, limit Limit) (res FarmResult, err error) {
	query := farmParams(filter, limit)
	req, err := http.Get(g.url(fmt.Sprintf("farms%s", query)))
	if err != nil {
		return
	}
	if req.StatusCode != http.StatusOK {
		err = parseError(req.Body)
		return
	}
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &res)
	return
}

func (g *GridProxyClientimpl) Node(nodeID uint32) (res NodeInfo, err error) {
	req, err := http.Get(g.url("nodes/%d", nodeID))
	if err != nil {
		return
	}
	if req.StatusCode != http.StatusOK {
		err = parseError(req.Body)
		return
	}
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &res)
	return
}

func (g *GridProxyClientimpl) NodeStatus(nodeID uint32) (res NodeStatus, err error) {
	req, err := http.Get(g.url("nodes/%d/status", nodeID))
	if err != nil {
		return
	}
	if req.StatusCode != http.StatusOK {
		err = parseError(req.Body)
		return
	}
	if err := json.NewDecoder(req.Body).Decode(&res); err != nil {
		return res, err
	}
	return
}
