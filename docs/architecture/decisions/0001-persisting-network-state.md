# 1. persisting_network_state

Date: 2022-11-10

## Status

Accepted

## Context

Network state should be stored somewhere since a network could span multiple deployments, and these deployments could be on the same node. 
Since terraform doesn't allow cyclic dependencies between its resources, it was decided that there should be a resource that allowed reading and writing data to.
The question is whether to use the kv store, or use a local state file.

## Decision

- Use a local state file to save network state, since the kv store could end up in an unclean state if the user decided to delete contracts manually from somewhere else than terraform.


## Consequences

- A user can now create multiple deployments on the same node, and vms would be assigned different local ips. (solving this [issue](https://github.com/threefoldtech/terraform-provider-grid/issues/11))