// Package provider is the terraform provider
package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
)

func dataSourceGatewayDomain() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Data source for computing gateway name proxy fqdn.",

		ReadContext: dataSourceGatewayRead,

		Schema: map[string]*schema.Schema{
			"node": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Node ID of the gateway",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the gateway name workload",
			},
			"fqdn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified domain name",
			},
		},
	}
}

// TODO: make this non failing
func dataSourceGatewayRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	nodeID := uint32(d.Get("node").(int))
	name := d.Get("name").(string)

	ncPool := client.NewNodeClientPool(tfPluginClient.RMB, tfPluginClient.RMBTimeout)
	nodeClient, err := ncPool.GetNodeClient(tfPluginClient.SubstrateConn, nodeID)
	if err != nil {
		return diag.FromErr(errors.Wrapf(err, "failed to get node client with ID %d", nodeID))
	}

	cfg, err := nodeClient.NetworkGetPublicConfig(ctx)
	if err != nil {
		return diag.FromErr(errors.Wrapf(err, "couldn't get node %d public config", nodeID))
	}

	if cfg.Domain == "" {
		return diag.FromErr(fmt.Errorf("node %d doesn't contain a domain in its public config", nodeID))
	}

	fqdn := fmt.Sprintf("%s.%s", name, cfg.Domain)
	err = d.Set("fqdn", fqdn)
	if err != nil {
		return diag.FromErr(errors.Wrapf(err, "couldn't set fqdn %s", fqdn))
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}
