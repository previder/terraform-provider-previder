# Previder Provider

The Previder provider is used to interact with resources on the Previder IaaS environment. 
The provider needs to be configured with an API token that will be provided by Previder.

Before using this README, make sure that you have installed the Previder Provider using INSTALL.

## Example Usage 
```
provider "previder" {
    token = "<token>"
}
```
## Argument reference
The following arguments are supported:
- token - (Required) This is your personal API token for accessing resources in the Previder IaaS environment.


## Resources
### previder_virtualnetwork

#### Example usage
```
resource "previder_virtualnetwork" "testlab-net" {
    name = "testlab-net"
    address_pool = [
        { 
            ip_start = "10.111.0.1" 
            ip_end = "10.111.0.253"
            ip_netmask = "255.255.255.0"
            ip_gateway = "10.111.0.254"
            ip_nameserver1 = "10.111.0.254"
            ip_nameserver2 = "10.111.0.254"
        }
    ]
}
```
#### Argument reference

The following arguments are supported:
- name : (Required) The network name
- address_pool : (Optional) Create a new pool of addresses after creating the network. 
    - ip_start : (Required) Specify the first usable network address.
    - ip_end : (Required) Specify the last usable network address.
    - ip_netmask : (Required)  Specify network mask ip address.
    - ip_gateway : (Optional) Specify gateway ip address.
    - ip_nameserver1 : (Optional) Specify nameserver ip address.
    - ip_nameserver2 : (Optional) Specify nameserver ip address. 


### previder_virtualmachine
#### Example usage 1
```
resource "previder_virtualmachine" "testlab-vm1" {
    name = "testlab-vm1"
    cpucores = 2
    memory = 1024
    template = "Ubuntu LTS"
    cluster = "Express"
    disk = [
        { size = 10240 }
    ]
    network_interface = [
        { network = "Public WAN" }
    ]
    user_data = <<EOF
#cloud-config

users:
  - name: ubuntu
    passwd: <base64 encrypted password>
EOF
    connection {
        user = "ubuntu"
        type = "ssh"
        timeout = "2m"
    }
}
```

#### Example usage 2
```
resource "previder_virtualmachine" "testlab-vm1" {
    name = "testlab-vm1"
    cpucores = 2
    memory = 1024
    template = "Ubuntu LTS"
    cluster = "Express"
    disk = [
        { size = 10240 }
    ]
    network_interface = [
        { network = "Public WAN" }
    ]
    depends_on = ["previder_virtualnetwork.www-net"]
    user_data = <<EOF
#cloud-config
ssh_authorized_keys:
  - "ssh-rsa <insert public key>"
  - "ssh-rsa <insert public key>"
EOF
    connection {
        user = "ubuntu"
        type = "ssh"
        timeout = "2m"
    }
}
```

#### Argument Reference
The following arguments are supported:
- name - (Required) 
- cpucores - (Required)
- memory - (Required)
- disk - (Required)
- cluster - (Optional)
- network_interface - (Required)
    - network - (Required)
- template - (Optional)
- user_data - (Optional)
- termination_protection - (Optional)

## Motivation

As projects besides e.g. the Previder Portal, the development team at Previder develops and maintains multiple projects aiming to integrate the previder IaaS environment.

## API Reference

This project uses the API client from the [previder-go-sdk](https://github.com/previder/previder-go-sdk) project.

## Contributors

* Check out the latest master to make sure the feature hasn't been implemented or the bug hasn't been fixed yet
* Fork the project
* Start a feature/bugfix branch
* Commit and push until you are happy with your contribution
* Send a merge request describing your exact problem, what and how you fixed it

