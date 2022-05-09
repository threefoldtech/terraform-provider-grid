package gridproxy

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

var (
	NodeExampleStr       = `{"id":"0000000510-000001-782e1","nodeId":1,"farmId":1,"twinId":9,"country":"Belgium","gridVersion":3,"city":"Unknown","uptime":1297882,"created":1649252220,"farmingPolicyId":1,"updatedAt":1650550422,"total_resources":{"cru":24,"sru":512110190592,"hru":9001778946048,"mru":202802933760},"used_resources":{"cru":52,"sru":793419710464,"hru":0,"mru":119957094400},"location":{"country":"Belgium","city":"Unknown"},"publicConfig":{"domain":"","gw4":"","gw6":"","ipv4":"","ipv6":""},"status":"up","certificationType":"Diy"}`
	NodesExampleStr      = fmt.Sprintf("[%s]", NodeExampleStr)
	FarmExampleStr       = `{"name":"Freefarm","farmId":1,"twinId":2,"pricingPolicyId":1,"stellarAddress":"","publicIps":[{"id":"0000001006-000001-f899f","ip":"185.206.122.35/24","farmId":"","contractId":142,"gateway":"185.206.122.1"},{"id":"0000001012-000001-23923","ip":"185.206.122.36/24","farmId":"","contractId":317,"gateway":"185.206.122.1"},{"id":"0000001019-000001-5001b","ip":"185.206.122.37/24","farmId":"","contractId":144,"gateway":"185.206.122.1"},{"id":"0000001070-000001-3e7e7","ip":"185.206.122.42/24","farmId":"","contractId":0,"gateway":"185.206.122.1"},{"id":"0000001047-000001-f6e0d","ip":"185.206.122.41/24","farmId":"","contractId":0,"gateway":"185.206.122.1"},{"id":"0000001042-000001-f65e8","ip":"185.206.122.40/24","farmId":"","contractId":0,"gateway":"185.206.122.1"},{"id":"0000000991-000001-aa42e","ip":"185.206.122.33/24","farmId":"","contractId":164,"gateway":"185.206.122.1"},{"id":"0000001037-000001-dad97","ip":"185.206.122.39/24","farmId":"","contractId":619,"gateway":"185.206.122.1"},{"id":"0000001075-000001-3b1ee","ip":"185.206.122.43/24","farmId":"","contractId":0,"gateway":"185.206.122.1"},{"id":"0000001084-000001-670af","ip":"185.206.122.44/24","farmId":"","contractId":0,"gateway":"185.206.122.1"},{"id":"0000001091-000001-c5b37","ip":"185.206.122.45/24","farmId":"","contractId":0,"gateway":"185.206.122.1"},{"id":"0000001096-000001-5f6c1","ip":"185.206.122.46/24","farmId":"","contractId":0,"gateway":"185.206.122.1"},{"id":"0000001101-000001-63193","ip":"185.206.122.47/24","farmId":"","contractId":0,"gateway":"185.206.122.1"},{"id":"0000001106-000001-c4f32","ip":"185.206.122.48/24","farmId":"","contractId":0,"gateway":"185.206.122.1"},{"id":"0000001168-000001-34245","ip":"185.206.122.49/24","farmId":"","contractId":0,"gateway":"185.206.122.1"},{"id":"0000001174-000001-db2a3","ip":"185.206.122.50/24","farmId":"","contractId":0,"gateway":"185.206.122.1"},{"id":"0000000999-000001-01080","ip":"185.206.122.34/24","farmId":"","contractId":677,"gateway":"185.206.122.1"},{"id":"0000001032-000001-5cfae","ip":"185.206.122.38/24","farmId":"","contractId":744,"gateway":"185.206.122.1"}]}`
	FarmsExampleStr      = fmt.Sprintf("[%s]", FarmExampleStr)
	NodeStatusExampleStr = `{"status":"up"}`

	NodeExample       = MarshalNode([]byte(NodeExampleStr))
	NodesExample      = []Node{NodeExample}
	NodeInfoExample   = MarshalNodeInfo([]byte(NodeExampleStr))
	FarmExample       = MarshalFarm([]byte(FarmExampleStr))
	FarmsExample      = []Farm{FarmExample}
	NodeStatusExample = MarshalNodeStatus([]byte(NodeStatusExampleStr))
)

func MustMarshal(data []byte, v interface{}) {
	if err := json.Unmarshal(data, v); err != nil {
		panic(err)
	}
}

func MarshalNodeInfo(data []byte) (info NodeInfo) {
	MustMarshal(data, &info)
	return
}

func MarshalNode(data []byte) (info Node) {
	MustMarshal(data, &info)
	return
}

func MarshalFarm(data []byte) (info Farm) {
	MustMarshal(data, &info)
	return
}

func MarshalNodeStatus(data []byte) (info NodeStatus) {
	MustMarshal(data, &info)
	return
}

type ProxyFunc func(url string) GridProxyClient

func TestConnectionFailures(t *testing.T) {
	testConnectionFailures(t, NewGridProxyClient)
}

func testConnectionFailures(t *testing.T, f ProxyFunc) {
	proxy := f("http://127.0.0.1:57854")
	endpoints := map[string]func() error{
		"ping": func() error {
			return proxy.Ping()
		},
		"nodes": func() error {
			_, err := proxy.Nodes(NodeFilter{}, Limit{})
			return err
		},
		"node": func() error {
			_, err := proxy.Node(1)
			return err
		},
		"farms": func() error {
			_, err := proxy.Farms(FarmFilter{}, Limit{})
			return err
		},
		"node_status": func() error {
			_, err := proxy.NodeStatus(1)
			return err
		},
	}
	for name, f := range endpoints {
		if f() == nil {
			t.Fatalf("proxy endpoint %s didn't fail for a connection-refused error", name)
		}
	}
}

func TestPingFailure(t *testing.T) {
	testPingFailure(t, NewGridProxyClient)
}

func testPingFailure(t *testing.T, f ProxyFunc) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`
		{
			"error": "some generic error"
		}
	`))
	}))
	defer ts.Close()

	proxy := f(ts.URL)
	err := proxy.Ping()
	if err == nil {
		t.Fatal("ping didn't fail for a status code error")
	}
}

func TestStatusCodeFailures(t *testing.T) {
	testStatusCodeFailures(t, NewGridProxyClient)
}

func testStatusCodeFailures(t *testing.T, f ProxyFunc) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`
			{
				"error": "some generic error"
			}
		`))
	}))
	defer ts.Close()
	proxy := f(ts.URL)
	endpoints := map[string]func() error{
		"nodes": func() error {
			_, err := proxy.Nodes(NodeFilter{}, Limit{})
			return err
		},
		"node": func() error {
			_, err := proxy.Node(1)
			return err
		},
		"farms": func() error {
			_, err := proxy.Farms(FarmFilter{}, Limit{})
			return err
		},
		"node_status": func() error {
			_, err := proxy.NodeStatus(1)
			return err
		},
	}
	for name, f := range endpoints {
		err := f()
		if err == nil {
			t.Fatalf("proxy endpoint %s didn't fail for a status code error", name)
		}
		if err.Error() != "some generic error" {
			t.Fatalf("error parsed incorrectly in %s: %s, should be: some generic error", name, err.Error())
		}
	}
}

func AssertHTTPRequest(
	t *testing.T,
	f ProxyFunc,
	method string,
	path string,
	response string,
	call func(proxy GridProxyClient) error,
) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedURL := r.URL.Path
		if r.URL.RawQuery != "" {
			expectedURL = fmt.Sprintf("%s?%s", expectedURL, r.URL.RawQuery)
		}
		if expectedURL == path && r.Method == method {
			w.WriteHeader(http.StatusOK)
			// panic(response)
			w.Write([]byte(response))
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"error": "expected path and methods: %s, %s. found: %s, %s"}`, path, method, expectedURL, r.Method)))
		}
	}))
	defer ts.Close()
	proxy := f(ts.URL)
	err := call(proxy)
	if err != nil {
		log.Printf(
			`
			path: %s
			response: %s
			`,
			path,
			response,
		)
		t.Fatal(err.Error())
	}
}

func TestSuccess(t *testing.T) {
	testSuccess(t, NewGridProxyClient)
}
func testSuccess(t *testing.T, f ProxyFunc) {
	nodesFilter, nodesLimit, expectedNodesURL := nodesFilterValues()
	farmsFilter, farmsLimit, expectedFarmsURL := farmsFilterValues()
	endpoints := map[string]struct {
		method   string
		path     string
		response string
		call     func(proxy GridProxyClient) error
	}{
		"nodes": {
			method:   "GET",
			path:     fmt.Sprintf("/nodes%s", expectedNodesURL),
			response: NodesExampleStr,
			call: func(proxy GridProxyClient) error {
				res, err := proxy.Nodes(nodesFilter, nodesLimit)
				if err != nil {
					return err
				}
				if !reflect.DeepEqual(NodesExample, res) {
					return fmt.Errorf("result mismatch: expected: %v, found: %v", NodesExample, res)
				}
				return nil
			},
		},
		"node": {
			method:   "GET",
			path:     "/nodes/1",
			response: NodeExampleStr,
			call: func(proxy GridProxyClient) error {
				res, err := proxy.Node(1)
				if err != nil {
					return err
				}
				if !reflect.DeepEqual(NodeInfoExample, res) {
					return fmt.Errorf("result mismatch: expected: %v, found: %v", NodeInfoExample, res)
				}
				return nil
			},
		},
		"farms": {
			method:   "GET",
			path:     fmt.Sprintf("/farms%s", expectedFarmsURL),
			response: FarmsExampleStr,
			call: func(proxy GridProxyClient) error {
				res, err := proxy.Farms(farmsFilter, farmsLimit)
				if err != nil {
					return err
				}
				if !reflect.DeepEqual(FarmsExample, res) {
					return fmt.Errorf("result mismatch: expected: %v, found: %v", NodeExample, res[0])
				}
				return nil
			},
		},
		"node_status": {
			method:   "GET",
			path:     "/nodes/1/status",
			response: NodeStatusExampleStr,
			call: func(proxy GridProxyClient) error {
				res, err := proxy.NodeStatus(1)
				if err != nil {
					return err
				}
				if !reflect.DeepEqual(NodeStatusExample, res) {
					return fmt.Errorf("result mismatch: expected: %v, found: %v", NodeStatusExample, res)
				}
				return nil
			},
		},
	}
	for _, endpoint := range endpoints {
		AssertHTTPRequest(t, f, endpoint.method, endpoint.path, endpoint.response, endpoint.call)
	}
}
