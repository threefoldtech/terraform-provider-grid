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
	client "github.com/threefoldtech/terraform-provider-grid/internal/node"
)

func dataSourceGatewayDomain() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Data source for computing gateway name proxy fqdn.",

		ReadContext: dataSourceGatewayRead,

		Schema: map[string]*schema.Schema{
			"node_id": {
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
	apiClient, ok := meta.(*apiClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into api client"))
	}

	nodeID := uint32(d.Get("node_id").(int))
	name := d.Get("name").(string)
	ncPool := client.NewNodeClientPool(apiClient.rmb)
	nodeClient, err := ncPool.GetNodeClient(apiClient.substrateConn, nodeID)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to get node client"))
	}
	ctx2, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	cfg, err := nodeClient.NetworkGetPublicConfig(ctx2)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't get node public config"))
	}
	if cfg.Domain == "" {
		return diag.FromErr(errors.New("node doesn't contain a domain in its public config"))
	}
	fqdn := fmt.Sprintf("%s.%s", name, cfg.Domain)
	err = d.Set("fqdn", fqdn)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't set fqdn"))
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}
