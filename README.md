# Previder Provider

The Previder provider is used to interact with resources on the Previder IaaS environment. 
The provider needs to be configured with an API token that will be provided by Previder.

Before using this README, make sure that you have installed the Previder Provider using INSTALL.


## Important notice

This Terraform/OpenTofu provider is provided to you "as-is" and without warranty of any kind, express, implied or otherwise, including without limitation, any warranty of fitness for a particular purpose.

If you have any questions, we encourage you to do your own research, seek out experts, and discuss with your community.
If there are questions that remain unanswered, please send an e-mail to managed@previder.nl. We’re going to do our best to help answer the questions that you have. Since the Terraform/OpenTofu provider is provided for free, please understand that more complex questions can only be answered for a fee.

## Example Usage 
```
terraform {
  required_providers {
    previder = {
      source  = "previder/previder"
      version = "~> 1.0"
    }
  }
}

provider "previder" {
    token = "<token>"
}
```
## Argument reference
The following arguments are supported:
- token - (Required) This is your personal API token for accessing resources in the Previder IaaS environment.
- customer - (Optional) For a default sub customer account to perform actions in

## Resources
### previder_virtual_network

#### Example usage
```
resource "previder_virtual_network" "testlab-net" {
    name = "testlab-net"
}
```
#### Argument reference

The following arguments are supported:
- name : (Required) The network name

### previder_virtual_server
#### Example usage 1
```
resource "previder_virtual_server" "test" {
  name = "test"
  cpu_cores = 2
  memory = 4096
  network_interfaces = {
    "NIC1" = {
      network = "56655e04e4b0069fba0c6252"
    },
    "NIC2" = {
      network = "5a17da71fcaae44a910027a9"
    }
  }
  guest_id = "other4xLinux64Guest"
  compute_cluster = "express-pdc2"
  disks = {
    "Disk1" = {
      size = 4096
    },
    "Disk2" = {
      size = 4096
    }
  }
  tags = [
    "tag1",
    "tag2"
  ]
}

```

#### Argument Reference
The following arguments are supported:
- name - (Required) 
- cpu_cores - (Required)
- cpu_sockets - (Optional)
- memory - (Required)
- disks - (Required) - The disks are always handled alphabetically!
- compute_cluster - (Optional)
- group - (Optional) This identifier can be found in the Previder Portal as ObjectId, or through the Previder API. 
- network_interfaces - (Required) The network_interfaces are always handled alphabetically!
    - network - (Required) This identifier can be found in the Previder Portal as ObjectId, or through the Previder API.
- template - (Optional) One of the fields template, guest_id or source_virtual_machine is required. 
- guest_id - (Optional)
- source_virtual_machine - (Optional)
- user_data - (Optional)
- termination_protection - (Optional)


### previder_kubernetes_cluster
#### Example usage
```
resource previder_kubernetes_cluster "testcluster" {
  name = "Tofu"
  cni = "cilium"
  network = "LocalNetIpBlock"
  vips = ["10.5.0.3"]
  version = "1.30.0"
  auto_update = false
  minimal_nodes = 1
  control_plane_cpu_cores = 2
  control_plane_memory_gb = 2
  control_plane_storage_gb = 25
  node_cpu_cores = 4
  node_memory_gb = 8
  node_storage_gb = 30
  compute_cluster = "express"
  high_available_control_plane = false
}
```

#### Argument Reference
The following arguments are supported:
- name (Required)
- cni (Optional) - Leave empty if you want to install your own CNI, or choose a supported CNI from the Previder Portal
- network (Required)
- vips (Required) - VIP(s) in your own network to reach the cluster 
- endpoints (Optional) - Optional extra SANs with which your cluster certificates will be reachable
- version (Optional) - Only optional when auto_update is false
- auto_update (Optional) - Only Required when version is not set
- auto_scale_enabled (Optional)
- minimal_nodes (Required) - You default number of nodes
- maximal_nodes (Optional) - Only required when auto_scale_enabled is true
- control_plane_cpu_cores (Required)
- control_plane_memory_gb (Required)
- control_plane_storage_gb (Required)
- node_cpu_cores (Required)
- node_memory_gb (Required)
- node_storage_gb (Required)
- compute_cluster (Required) - When set to a global cluster like "express", the nodes will automatically be spread over locations. When choosing a specific location, all nodes will run in that location.
- high_available_control_plane (Optional) - when set to true, 3 control plane nodes will be deployed instead of 1

## Motivation

As projects besides e.g. the Previder Portal, the development team at Previder develops and maintains multiple projects aiming to integrate the Previder IaaS environment.

## API Reference

This project uses the API client from the [previder-go-sdk](https://github.com/previder/previder-go-sdk) project.

## Contributors

* Check out the latest master to make sure the feature has not been implemented or the bug is not fixed yet
* Fork the project
* Start a feature/bugfix branch
* Commit and push until you are happy with your contribution
* Send a pull request describing your exact problem, what and how you fixed it

