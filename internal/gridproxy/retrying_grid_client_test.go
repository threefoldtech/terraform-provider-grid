package gridproxy

import (
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
)

type requestCounter struct {
	Counter int
}

func NewRequestCounter() GridProxyClient {
	return &requestCounter{0}
}

func (r *requestCounter) Ping() error {
	r.Counter++
	return errors.New("error")
}
func (r *requestCounter) Nodes(filter NodeFilter, pagination Limit) (res []Node, err error) {
	r.Counter++
	return nil, errors.New("error")
}
func (r *requestCounter) Farms(filter FarmFilter, pagination Limit) (res FarmResult, err error) {
	r.Counter++
	return nil, errors.New("error")
}
func (r *requestCounter) Node(nodeID uint32) (res NodeInfo, err error) {
	r.Counter++
	return NodeInfo{}, errors.New("error")
}
func (r *requestCounter) NodeStatus(nodeID uint32) (res NodeStatus, err error) {
	r.Counter++
	return NodeStatus{}, errors.New("error")
}

func retryingConstructor(u string) GridProxyClient {
	return NewRetryingGridProxyClientWithTimeout(NewGridProxyClient(u), 1*time.Millisecond)
}

func TestRetryingConnectionFailures(t *testing.T) {
	testConnectionFailures(t, retryingConstructor)
}

func TestRetryingPingFailure(t *testing.T) {
	testPingFailure(t, retryingConstructor)
}

func TestRetryingStatusCodeFailures(t *testing.T) {
	testStatusCodeFailures(t, retryingConstructor)
}

func TestRetryingSuccess(t *testing.T) {
	testSuccess(t, retryingConstructor)
}

func TestCalledMultipleTimes(t *testing.T) {
	r := NewRequestCounter()
	proxy := NewRetryingGridProxyClientWithTimeout(r, 1*time.Millisecond)
	methods := map[string]func(){
		"nodes": func() {
			proxy.Nodes(NodeFilter{}, Limit{})
		},
		"node": func() {
			proxy.Node(1)
		},
		"farms": func() {
			proxy.Farms(FarmFilter{}, Limit{})
		},
		"node_status": func() {
			proxy.NodeStatus(1)
		},
	}
	for endpoint, f := range methods {
		beforeCount := r.(*requestCounter).Counter
		f()
		afterCount := r.(*requestCounter).Counter
		fmt.Printf("%d %d ", beforeCount, afterCount)
		if afterCount-beforeCount <= 1 {
			t.Fatalf("retrying %s client is expected to try more than once. before calls: %d, after calls: %d", endpoint, beforeCount, afterCount)
		}
	}
}
