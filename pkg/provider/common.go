package provider

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	gormb "github.com/threefoldtech/go-rmb"
	client "github.com/threefoldtech/terraform-provider-grid/pkg/node"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

const RMB_WORKERS = 10

func startRmbIfNeeded(ctx context.Context, api *apiClient) {
	if api.use_rmb_proxy {
		return
	}
	rmbClient, err := gormb.NewServer(api.manager, "127.0.0.1:6379", RMB_WORKERS, api.identity)
	if err != nil {
		log.Fatalf("couldn't start server %s\n", err)
	}
	if err := rmbClient.Serve(ctx, api.manager); err != nil {
		log.Printf("error serving rmb %s\n", err)
	}
}

func flistChecksumURL(url string) string {
	return fmt.Sprintf("%s.md5", url)
}
func getFlistChecksum(url string) (string, error) {
	response, err := http.Get(flistChecksumURL(url))
	if err != nil {
		return "", err
	}
	hash, err := io.ReadAll(response.Body)
	return strings.TrimSpace(string(hash)), err
}

func isNodeUp(ctx context.Context, nc *client.NodeClient) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := nc.NetworkListInterfaces(ctx)
	if err != nil {
		return err
	}

	return nil
}

func isNodesUp(ctx context.Context, sub subi.SubstrateExt, nodes []uint32, nc client.NodeClientCollection) error {
	for _, node := range nodes {
		cl, err := nc.GetNodeClient(sub, node)
		if err != nil {
			return fmt.Errorf("couldn't get node %d client: %w", node, err)
		}
		if err := isNodeUp(ctx, cl); err != nil {
			return fmt.Errorf("couldn't reach node %d: %w", node, err)
		}
	}

	return nil
}
func constructPublicIPWorkload(workloadName string, ipv4 bool, ipv6 bool) gridtypes.Workload {
	return gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name(workloadName),
		Type:    zos.PublicIPType,
		Data: gridtypes.MustMarshal(zos.PublicIP{
			V4: ipv4,
			V6: ipv6,
		}),
	}
}
