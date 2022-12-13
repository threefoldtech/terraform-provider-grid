### This is an attempt to explain the design of the grid provider.

## This should be split into two parts: Before v1.6.3, and from v1.6.3

- Before v2.0.0:
    - There is one provider used across all nets (dev, test, qa, main), the only change is the chain interaction using substrate-client, and a user can specify which chain they want to interact with. 
    - Terraform by design requires implementing four functions for every resource besides its schema to handle CRUD operations for this resource.
    - Deployment, gateway_name, and gateway_fqdn resources use the same flow:
        - a wrapper middleware function wraps every CRUD operation. This wrapper implements validation and state persistence, so that these processes are guarenteed to run with any CRUD operation.
    - Network resource use the following flow:
        - Each CRUD operation handles it's own logic, with no middleware included.
        - A network workload is deployed on each node provided by the user, and each node has some or all other nodes as peers.
        - If the user requires wg access, a node with a public config is used as an access node. If the nodes provided has a node with public config, it is used as an access node, if not, a random eligible node is used.
    - K8s resource use the following flow:
        - Each CRUD operation handles it's own logic, with no middleware included.
        - It's basically regular Deployment resources, but with a special flist for kubernetes, and with some enironment variables set for each vm.

- From v2.0.0:
    - capacity reservation was introduces in v2.0.0, so some schema changes had to be done:
        - users now need to first create a capacity reservation contract to use it in deploying resources.
        - a user now cannot decide which node they want to deploy on, they can only use their capacity reservation contract id then the deployer will deploy this deployment on the node that has the capacity reservation.
    - since the new provider will not be backward compatible, a user could not use this provider for all nets, and will have to decide which specific version of the provider they want to use.