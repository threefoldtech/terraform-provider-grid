// Package provider is the terraform provider
package provider

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "Resource to deploy a network on the grid. This is a private wireguard network. A user could specify that they want to have a user access endpoint to this network through the `add_wg_access` flag. A separate workload is deployed on each of the specified nodes, with the peers for each workload configured in a way making any pair of nodes in the network accessible to each other.",

		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		UpdateContext: resourceNetworkUpdate,
		DeleteContext: resourceNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Network workloads Name.  This has to be unique within the node.",
			},
			"solution_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Solution type for created contract to be consistent across threefold tooling.",
				Default:     "Network",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description of the network workloads.",
			},
			"nodes": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
				Description: "List of node ids to add to the network.",
			},
			"ip_range": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Network IP range (e.g. 10.1.2.0/16). Has to have a subnet mask of 16.",
			},
			"add_wg_access": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Flag to generate wireguard configuration for external user access to the network.",
			},
			"access_wg_config": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Generated wireguard configuration for external user access to the network.",
			},
			"external_ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Wireguard IP assigned for external user access.",
			},
			"external_sk": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "External user private key used in encryption while communicating through Wireguard network.",
			},
			"public_node_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Public node id (in case it's added). Used for wireguard access and supporting hidden nodes.",
			},
			"nodes_ip_range": {
				Type:        schema.TypeMap,
				Computed:    true,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Computed values of nodes' IP ranges after deployment.",
			},
			"node_deployment_id": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: "Mapping from each node to its deployment id.",
			},
		},
	}
}

// NewNetwork reads the network resource configuration data from schema.ResourceData, converts them into a network instance, and returns this instance.
func newNetwork(d *schema.ResourceData) (*workloads.ZNet, error) {
	var err error
	nodesIf := d.Get("nodes").([]interface{})
	nodes := make([]uint32, len(nodesIf))
	for idx, n := range nodesIf {
		nodes[idx] = uint32(n.(int))
	}

	nodeDeploymentIDIf := d.Get("node_deployment_id").(map[string]interface{})
	nodeDeploymentID := make(map[uint32]uint64)
	for node, id := range nodeDeploymentIDIf {
		nodeInt, err := strconv.ParseUint(node, 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't parse node id '%s'", node)
		}
		deploymentID := uint64(id.(int))
		nodeDeploymentID[uint32(nodeInt)] = deploymentID
	}

	nodesIPRange := make(map[uint32]gridtypes.IPNet)
	nodesIPRangeIf := d.Get("nodes_ip_range").(map[string]interface{})
	for node, r := range nodesIPRangeIf {
		nodeInt, err := strconv.ParseUint(node, 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't parse node id '%s'", node)
		}
		nodesIPRange[uint32(nodeInt)], err = gridtypes.ParseIPNet(r.(string))
		if err != nil {
			return nil, errors.Wrap(err, "couldn't parse node ip range")
		}
	}

	// external node related data
	addWGAccess := d.Get("add_wg_access").(bool)

	var externalIP *gridtypes.IPNet
	externalIPStr := d.Get("external_ip").(string)
	if externalIPStr != "" {
		ip, err := gridtypes.ParseIPNet(externalIPStr)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't parse external ip")
		}
		externalIP = &ip
	}

	var externalSK wgtypes.Key
	if d.Get("external_sk").(string) != "" {
		externalSK, err = wgtypes.ParseKey(d.Get("external_sk").(string))
	} else {
		externalSK, err = wgtypes.GeneratePrivateKey()
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get external_sk key")
	}

	ipRange, err := gridtypes.ParseIPNet(d.Get("ip_range").(string))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse network ip range")
	}

	znet := workloads.ZNet{
		Name:             d.Get("name").(string),
		Description:      d.Get("description").(string),
		Nodes:            nodes,
		IPRange:          ipRange,
		AddWGAccess:      addWGAccess,
		AccessWGConfig:   d.Get("access_wg_config").(string),
		ExternalIP:       externalIP,
		ExternalSK:       externalSK,
		PublicNodeID:     uint32(d.Get("public_node_id").(int)),
		NodesIPRange:     nodesIPRange,
		NodeDeploymentID: nodeDeploymentID,
		Keys:             make(map[uint32]wgtypes.Key),
		WGPort:           make(map[uint32]int),
	}

	return &znet, nil
}

func storeState(d *schema.ResourceData, tfPluginClient *deployer.TFPluginClient, net *workloads.ZNet) (errors error) {

	nodeDeploymentID := make(map[string]interface{})
	for node, id := range net.NodeDeploymentID {
		nodeDeploymentID[fmt.Sprintf("%d", node)] = int(id)
	}

	nodesIPRange := make(map[string]interface{})
	for node, r := range net.NodesIPRange {
		nodesIPRange[fmt.Sprintf("%d", node)] = r.String()
	}

	nodes := make([]uint32, 0)
	for _, node := range net.Nodes {
		if _, ok := net.NodeDeploymentID[node]; ok {
			nodes = append(nodes, node)
		}
	}

	for node := range net.NodeDeploymentID {
		if !workloads.Contains(nodes, node) {
			if net.PublicNodeID == node {
				continue
			}
			nodes = append(nodes, node)
		}
	}
	log.Printf("setting deployer object nodes: %v", nodes)
	// update network local status
	updateNetworkLocalState(tfPluginClient, net)

	net.Nodes = nodes

	log.Printf("storing nodes: : %v", nodes)
	err := d.Set("nodes", nodes)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("ip_range", net.IPRange.String())
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("access_wg_config", net.AccessWGConfig)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	if net.ExternalIP == nil {
		err = d.Set("external_ip", nil)
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	} else {
		err = d.Set("external_ip", net.ExternalIP.String())
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	err = d.Set("external_sk", net.ExternalSK.String())
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("public_node_id", net.PublicNodeID)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	// plural or singular?
	err = d.Set("nodes_ip_range", nodesIPRange)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("node_deployment_id", nodeDeploymentID)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	return
}

func updateNetworkLocalState(tfPluginClient *deployer.TFPluginClient, net *workloads.ZNet) {
	ns := tfPluginClient.State.GetNetworks()
	ns.DeleteNetwork(net.Name)
	network := ns.GetNetwork(net.Name)
	for nodeID, subnet := range net.NodesIPRange {
		network.SetNodeSubnet(nodeID, subnet.String())
	}
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	net, err := newNetwork(d)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load network data"))
	}

	err = tfPluginClient.NetworkDeployer.Deploy(ctx, net)
	if err != nil {
		if len(net.NodeDeploymentID) != 0 {
			// failed to deploy and failed to revert, store the current state locally
			diags = diag.FromErr(err)
		} else {
			return diag.FromErr(err)
		}
	}

	err = storeState(d, tfPluginClient, net)
	if err != nil {
		diags = diag.FromErr(err)
	}

	d.SetId(uuid.New().String())
	return diags
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	net, err := newNetwork(d)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load network data"))
	}

	err = tfPluginClient.NetworkDeployer.Deploy(ctx, net)
	if err != nil {
		diags = diag.FromErr(err)
	}

	err = storeState(d, tfPluginClient, net)
	if err != nil {
		diags = diag.FromErr(err)
	}

	return diags
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	net, err := newNetwork(d)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load network data"))
	}

	if err := tfPluginClient.NetworkDeployer.InvalidateBrokenAttributes(net); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't invalidate broken attributes"))
	}

	err = tfPluginClient.NetworkDeployer.ReadNodesConfig(ctx, net)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  errTerraformOutSync,
			Detail:   err.Error(),
		})
		return diags
	}

	err = storeState(d, tfPluginClient, net)
	if err != nil {
		diags = diag.FromErr(err)
	}

	return diags
}

func resourceNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	net, err := newNetwork(d)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load network data"))
	}

	err = tfPluginClient.NetworkDeployer.Cancel(ctx, net)
	if err != nil {
		diags = diag.FromErr(err)
	}

	if err == nil {
		d.SetId("")
	} else {
		err = storeState(d, tfPluginClient, net)
		if err != nil {
			diags = diag.FromErr(err)
		}
	}
	return diags
}
