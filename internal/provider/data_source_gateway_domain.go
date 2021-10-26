package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

func dataSourceGatewayDomain() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Data source for computing gateway name proxy fqdn.",

		ReadContext: dataSourceGatewayRead,

		Schema: map[string]*schema.Schema{
			"node": {
				Description: "Node ID of the gateway",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"name": {
				Description: "The name ",
				Type:        schema.TypeString,
				Required:    true,
			},
			"fqdn": {
				Description: "Full domain name",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceGatewayRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiClient := meta.(*apiClient)
	go startRmbIfNeeded(ctx, apiClient)
	nodeID := uint32(d.Get("node").(int))
	name := d.Get("name").(string)
	ncPool := NewNodeClient(apiClient.sub, apiClient.rmb)
	nodeClient, err := ncPool.getNodeClient(nodeID)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "failed to get node client"))
	}
	sub, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	cfg, err := nodeClient.NetworkGetPublicConfig(sub)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't get node public config"))
	}
	if cfg.Domain == "" {
		return diag.FromErr(errors.New("node doesn't contain a domain in its public config"))
	}
	fqdn := fmt.Sprintf("%s.%s", name, cfg.Domain)
	d.Set("fqdn", fqdn)
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}
