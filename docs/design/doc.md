### This is an attempt to explain the design of the grid provider.

## This should be split into two parts: Before v2.0.0, and from v2.0.0

- Before v2.0.0:
    - There is one provider used across all nets (dev, test, qa, main), the only change is the chain interaction using substrate-client, and a user can specify which chain they want to interact with. 
    - Terraform by design requires implementing four functions for every resource, besides its schema, to handle CRUD operations for this resource.
    - The internals of each resource is described in the following sections:
        - # Deployment resource:
            - a wrapper middleware function wraps every CRUD operation. This wrapper implements validation and state persistence, so that these processes are guaranteed to run with any CRUD operation.
            - for every CRUD operation in the deployment resource, a DeploymentDeployer object is initialized, using the data from the terraform state. This object has the needed information to from the user to create a grid deployment consisting of vms, zdbs, qsfss, or disks. This deployment should belong to a network, which is also provider in the DeploymentDeployer object. Then, the ConstructVersionlessDeployments function is used to generate a grid.Deployment object, which is then used by the deployer package to be deployed on the grid.

        - TODO: add gateways docs.
        - # Network resource:
            - Each CRUD operation handles it's own logic, with no middleware included.
            - for every CRUD operation, a NetworkDeployer object is initialized, using the data from the terraform state. The NetworkDeployer has information about which nodes are included in the network, which ports are used, whether or not an access node is used (depending on whether user required wg access, or if there was a node with no public configuration included in the network), the deployment ids for each network workload on each node, the public keys for each node, the ip range of each node, and the user access ip and private key (only if user required wg access).
            - using the NetworkDeployer, a network workload is deployed on each node provided by the user, and each node has some or all other nodes as peers.
            - If the user requires wg access, a node with a public config is used as an access node. If the nodes provided has a node with public config, it is used as the access node, if not, a random eligible node is used.

        - # K8s resource:
            - Each CRUD operation handles it's own logic, with no middleware included.
            - It's basically regular Deployment resources, but with a special flist for kubernetes, and with some environment variables set for each vm.
            - for every CRUD operation, a K8sDeployer object is initialized, using the data from the terraform state. The K8sDeployer has information about the master node and the workers nodes (on which grid node it is deployed, and how much resources are assigned for them), the network name that this k8s cluster will belong to, a token that is used by other workers to join the cluster, an ssh key to be added to the authorized keys on each node on the cluster, and the deployment ids for each deployment on each node.

    - The provider uses a few local packages to assist in the deployment process:
        - # Deployer package:
            - All resources then use the deployer package to perform the desired CRUD operations on the grid. The deployer package compares between the old state and the new state in the following way:
                - if a deployment is in the old state but not in the new state, then it needs to be deleted.
                - if a deployment is in the new state but not in the old state, then it needs to be created.
                - if a deployment is in both states, then it needs to be updates.
            - If for some reason a failure happens while deploying, the deployer tries to revert the changes it did, if some failure happens while reverting, it is reported to the user.
            - the deployer uses the substrate-client package, to make requests to the chain.
        
        - # Subi package:
            - users can use the provider on their desired net (dev, test, qa, or main) by specifying it in the state, or using environment variables.
            - [substrate-client](https://github.com/threefoldtech/substrate-client) is used to make requests to the chain, with each version of the client capable with speaking to some or all nets.
            - the subi package provides a single interface that every version of the substrate-client should conform to.
            - this interface is then used by the deployer while interacting with the chain.

        - # State package:
            - terraform does a great job handling resources states, and keeps a sence of sorting between resources deployment. 
            - to do this, each resource has to have its own state, and the flow of information between states has to be unidirectional (no two resources can import data from each other).
            - TODO: complete state package description.

- From v2.0.0:
    - capacity reservation was introduced in v2.0.0, so some schema changes had to be done:
        - users now need to first create a capacity reservation contract to use it in deploying resources.
        - a user now cannot decide which node they want to deploy on, they can only use their capacity reservation contract id then the deployer will deploy this deployment on the node that has the capacity reservation.
    - since the new provider will not be backward compatible, a user could not use this provider for all nets, and will have to decide which specific version of the provider they want to use.