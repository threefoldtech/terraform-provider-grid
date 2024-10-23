// Package provider is the terraform provider
package provider

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"slices"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
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
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Network workloads Name.  This has to be unique within the node. Must contain only alphanumeric and underscore characters.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile(nameValidationRegex), nameValidationErrorMessage)),
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
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Network IP range (e.g. 10.1.2.0/16). Has to have a subnet mask of 16.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsCIDRNetwork(16, 16)),
			},
			"mycelium_keys": {
				Type:        schema.TypeMap,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Network mycelium keys per node (e.g. 9751c596c7c951aedad1a5f78f18b59515064adf660e0d55abead65e6fbbd627). Hex encoded 32 bytes.",
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
func newNetwork(ctx context.Context, d *schema.ResourceData, ncPool client.NodeClientGetter, sub subi.SubstrateExt) (workloads.Network, error) {
	var light bool
	var err error

	nodesIf := d.Get("nodes").([]interface{})
	nodes := make([]uint32, len(nodesIf))
	for idx, n := range nodesIf {
		nodes[idx] = uint32(n.(int))
	}

	var nodesLight []bool
	for _, n := range nodes {
		isLight, err := isZosLight(ctx, n, ncPool, sub)
		if err != nil {
			return nil, err
		}

		nodesLight = append(nodesLight, isLight)
	}

	// if no nodes requires to use network version 4 then it is a light network
	if !slices.Contains(nodesLight, false) {
		light = true
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

	nodesIPRange := make(map[uint32]zos.IPNet)
	nodesIPRangeIf := d.Get("nodes_ip_range").(map[string]interface{})
	for node, r := range nodesIPRangeIf {
		nodeInt, err := strconv.ParseUint(node, 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't parse node id '%s'", node)
		}
		nodesIPRange[uint32(nodeInt)], err = zos.ParseIPNet(r.(string))
		if err != nil {
			return nil, errors.Wrap(err, "couldn't parse node ip range")
		}
	}

	myceliumKeysIf := d.Get("mycelium_keys").(map[string]interface{})
	myceliumKeys := make(map[uint32][]byte)
	for node, key := range myceliumKeysIf {
		nodeID, err := strconv.ParseUint(node, 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't parse node id '%s'", node)
		}

		myceliumKey, err := hex.DecodeString(key.(string))
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't decode mycelium key '%s'", key)
		}

		myceliumKeys[uint32(nodeID)] = myceliumKey
	}

	// external node related data
	addWGAccess := d.Get("add_wg_access").(bool)

	var externalIP *zos.IPNet
	externalIPStr := d.Get("external_ip").(string)
	if externalIPStr != "" {
		ip, err := zos.ParseIPNet(externalIPStr)
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

	ipRange, err := zos.ParseIPNet(d.Get("ip_range").(string))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse network ip range")
	}

	if light {
		return &workloads.ZNetLight{
			Name:             d.Get("name").(string),
			Description:      d.Get("description").(string),
			Nodes:            nodes,
			IPRange:          ipRange,
			MyceliumKeys:     myceliumKeys,
			PublicNodeID:     uint32(d.Get("public_node_id").(int)),
			NodesIPRange:     nodesIPRange,
			NodeDeploymentID: nodeDeploymentID,
		}, nil
	}

	return &workloads.ZNet{
		Name:             d.Get("name").(string),
		Description:      d.Get("description").(string),
		Nodes:            nodes,
		IPRange:          ipRange,
		MyceliumKeys:     myceliumKeys,
		AddWGAccess:      addWGAccess,
		AccessWGConfig:   d.Get("access_wg_config").(string),
		ExternalIP:       externalIP,
		ExternalSK:       externalSK,
		PublicNodeID:     uint32(d.Get("public_node_id").(int)),
		NodesIPRange:     nodesIPRange,
		NodeDeploymentID: nodeDeploymentID,
		Keys:             make(map[uint32]wgtypes.Key),
		WGPort:           make(map[uint32]int),
	}, nil
}

func isZosLight(ctx context.Context, nodeID uint32, ncPool client.NodeClientGetter, sub subi.SubstrateExt) (bool, error) {
	nodeClient, err := ncPool.GetNodeClient(sub, nodeID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get node client '%d'", nodeID)
	}

	features, err := nodeClient.SystemGetNodeFeatures(ctx)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get node features '%d'", nodeID)
	}

	return slices.Contains(features, zos.NetworkLightType), nil
}

func storeState(d *schema.ResourceData, tfPluginClient *deployer.TFPluginClient, net workloads.Network) (errors error) {
	nodeDeploymentID := make(map[string]interface{})
	for node, id := range net.GetNodeDeploymentID() {
		nodeDeploymentID[fmt.Sprintf("%d", node)] = int(id)
	}

	nodesIPRange := make(map[string]interface{})
	for node, r := range net.GetNodesIPRange() {
		nodesIPRange[fmt.Sprintf("%d", node)] = r.String()
	}

	myceliumKeys := make(map[string]interface{})
	for node, key := range net.GetMyceliumKeys() {
		myceliumKeys[fmt.Sprintf("%d", node)] = hex.EncodeToString(key)
	}

	nodes := make([]uint32, 0)
	for _, node := range net.GetNodes() {
		if _, ok := net.GetNodeDeploymentID()[node]; ok {
			nodes = append(nodes, node)
		}
	}

	for node := range net.GetNodeDeploymentID() {
		if !workloads.Contains(nodes, node) {
			if net.GetPublicNodeID() == node {
				continue
			}
			nodes = append(nodes, node)
		}
	}
	log.Printf("setting deployer object nodes: %v", nodes)
	// update network local status
	updateNetworkLocalState(tfPluginClient, net)

	net.SetNodes(nodes)

	log.Printf("storing nodes: : %v", nodes)
	err := d.Set("nodes", nodes)
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("ip_range", net.GetIPRange().String())
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("access_wg_config", net.GetAccessWGConfig())
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	if net.GetExternalIP() == nil {
		err = d.Set("external_ip", nil)
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	} else {
		err = d.Set("external_ip", net.GetExternalIP().String())
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	err = d.Set("external_sk", net.GetExternalSK().String())
	if err != nil {
		errors = multierror.Append(errors, err)
	}

	err = d.Set("public_node_id", net.GetPublicNodeID())
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

func updateNetworkLocalState(tfPluginClient *deployer.TFPluginClient, net workloads.Network) {
	tfPluginClient.State.Networks.DeleteNetwork(net.GetName())
	tfPluginClient.State.Networks.UpdateNetworkSubnets(net.GetName(), net.GetNodesIPRange())
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	tfPluginClient, ok := meta.(*deployer.TFPluginClient)
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to cast meta into threefold plugin client"))
	}

	net, err := newNetwork(ctx, d, tfPluginClient.NcPool, tfPluginClient.SubstrateConn)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load network data"))
	}

	err = tfPluginClient.NetworkDeployer.Deploy(ctx, net)
	if err != nil {
		if len(net.GetNodeDeploymentID()) != 0 {
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

	net, err := newNetwork(ctx, d, tfPluginClient.NcPool, tfPluginClient.SubstrateConn)
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

	net, err := newNetwork(ctx, d, tfPluginClient.NcPool, tfPluginClient.SubstrateConn)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't load network data"))
	}

	if err := net.InvalidateBrokenAttributes(tfPluginClient.SubstrateConn, tfPluginClient.NcPool); err != nil {
		return diag.FromErr(errors.Wrap(err, "couldn't invalidate broken attributes"))
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

	net, err := newNetwork(ctx, d, tfPluginClient.NcPool, tfPluginClient.SubstrateConn)
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
