# Development guide (a vision not a reality)

This terraform plugin has a somewhat non-conventional nature from a terraform perspective. Resources in terraform in general should be a one-to-one mapping between a rest api object, and the CRUD operations are implemented for each of them. The plugin is different in that a single resource can be a mapping to a multiple resources (e.g. the network/k8s corresponds to a multiple remote deployment on the network's nodes).

A read operation always precedes an update or delete, this will be important when discussing the plugin CRUD implementation.

Another thing to consider is the terraform (weird) behavior on errors in the CRUD operations. They are listed below.

## Terraform plugin error handling

### Create

1. SetId(...) called and an error returned:

    The resource state is updated and marked as tainted (subsequent apply will cause delete/create instead of an update).
2. SetId(...) not called:

    The resource state is not updated and an error is displayed (the returned errors in diags or "Provider produced inconsistent result after apply" if non is returned)

### READ

When an error is returned, nothing is updated and the error is displayed.

### UPDATE

When an error is returned, all updates take place, the error is returned, and the object is not tainted.

### DELETE

1. When an error is returned, nothing is updated and the error is displayed.
2. If no error is returned, the resource is destroyed from the state.

# Plugin implementation details

Every resource correspond to a golang object (e.g. resource_network has a NetworkDeployer). Its properties are divided into 4 categories:

1. Identifiers: always a map from a node id to a deployment id
2. Inputs: The required/optional non-computed terraform arguments
3. Computed: The computed attributes (other than the identifiers) that is known after the deployment
4. Helpers: Properties that is neither needed by the user nor are kept track of in the state (e.g. wireguard ports in the network).

The real resource id is always set to a UUID since a deployment id is not guaranteed to stay as is after updates, and there are often multiple deployments.

It's assumed that the substrate is either fully down or functioning correctly. Misbehaving (like returning a 0 for an existing node id) is not handled (would be a nightmare).

A (very possible) scenario must be kept in mind while implementing the CRUD operations: a node in a stable network can be shutdown causing all interactions with it to result in failure. This shouldn't prevent the user from removing it from removing the node from the network.

## Read

Reading shouldn't raise an error in any case. This is to make sure an update/delete is possible when a node is down (the scenario above). So it's considered a best-effort method to sync the local state with the remote. It performs the following:

1. Remove the out of fund/deleted contracts.
2. Update the identifiers, the computed attributes, and the input attributes.
3. Return errors as warnings
This ensures a non-blocking feedback is returned to the user.

## Create

Its steps are as the following:

1. Try to create the requested deployments
2. In case of any failure, try to revert them
3. In case of a revert failure, set the id and store the active deployment ids.
4. If the revert succeeded, don't set the id and return the error
In case an id is set:

- The computed attributes MUST be updated. This is to ensure the output variables and downstream resources gets the correct values.
- The input attributes MAY be updated. It is not necessary though since an update/delte is always preceded by a read.

## Update

- It must remove the invalid identifiers to handle delete failures.
- Don't fetch data until you need it (don't get the info of a to-be-removed deployment).
- Assume the inputs is correct: This can be dangerous in case of a read failure, followed by an update. The input might not be in sync. But this would complicate the update handling and would cause the node-shutdown scenario to be not handled correctly

The attribute updates are handled like in the create (computed are a must/inputs are optional).

## Delete

Delete the contracts (fail if one failed), and try to call DeploymentDelete (not a requirement since the node will delete it anyway).
