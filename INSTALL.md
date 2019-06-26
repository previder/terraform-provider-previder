## Synopsis

Terraform is used to create, manage, and manipulate infrastructure resources. Examples of resources include physical machines, VMs, network switches, containers, etc.
Terraform is agnostic to the underlying platforms by supporting providers. A provider is responsible for understanding API interactions and exposing resources.<br />
The Previder Provider will connect to the Previder IaaS environment and deploy Virtual Servers and Virtual Networks.

## Code Example <a name="example"></a>

Using the following configuration files the Previder provider for TerraForm can be used to deploy and maintain infrastructures in the Previder Portal.
An API token is required, this can be acquired from the Previder Developers. With this token you will have admin privileges to you environment.

This example will deploy one subnet with one servers bound to it and one webserver connecting directly to the outside world ("Public WAN" network)<br />
Only the server is deployed, installing Apache or Nginx will be done by SSH commands, ansible, puppet, etc.

## Installation

The easy part of TerraForm is the installation of TerraForm itself. The project consists from one executable built in the GoLang language.<br />
Browse to the download page of TerraForm which is located at https://www.terraform.io/downloads.html<br />
Download the executable for your OS. The example uses Linux 64-bit. Other OSes could differ, but the mainline should stay the same.<br />
Extract the executable to the target directory. Latest version available at time of writing this manual is 0.12.3.

```bash
mkdir -p /opt/terraform
cd /opt/terraform
wget "https://releases.hashicorp.com/terraform/0.12.3/terraform_0.12.3_linux_amd64.zip" -O terraform.zip
unzip terraform.zip
./terraform version
```

After TerraForm has been downloaded and the version verified. The Provider can be cloned using Git and built.<br />
Make sure you have set the GOPATH environment variable and at least GoLang 1.12.6 available. This version is required by TerraForm.<br />

```bash
cd $GOPATH/src
git clone github.com/previder/terraform-provider-previder
cd terraform-provider-previder
go get
go build
```

This should result in a single executable file which can be used to connect to the Previder Portal.<br />
move the resulting file to /opt/terraform/providers/terraform-provider-previder

```bash
mkdir -p /opt/terraform/providers
mv $GOPATH/src/terraform-provider-previder/terraform-provider-previder /opt/terraform/providers/terraform-provider-previder
```

Using the configuration files shown in [the example](#Provider configuration), a base deployment of multiple resources is possible.

### Provider configuration
~/.terraformrc
```
providers {
  previder = "/opt/terraform/providers/terraform-provider-previder"
}
```


```
### Testing the configuration
Using the following command you may check if the configurations are parsed correctly.
```
./terraform plan
```
The output will show what TerraForm will be performing and should look similar to the output below:
```Diff
+ previder_virtualmachine.www-internal
    cluster:                "Express"
    cpucores:               "1"
    disksize:               "10240"
    memory:                 "1024"
    name:                   "www-internal"
    network:                "www-net"
    state:                  "<computed>"
    template:               "CoreOS"
    termination_protection: "<computed>"
    user_data:              "#cloud-config\n\nusers:\n  - name: core\n    passwd: <encrypted password>\n"

+ previder_virtualmachine.www-public
    cluster:                "Express"
    cpucores:               "1"
    disksize:               "10240"
    memory:                 "1024"
    name:                   "www-public"
    network:                "Public WAN"
    state:                  "<computed>"
    template:               "CoreOS"
    termination_protection: "<computed>"
    user_data:              "#cloud-config\n\nusers:\n  - name: core\n    passwd: <encrypted password>\n"

+ previder_virtualnetwork.www-net
    name: "www-net"

```

### Deploying
And apply the infrastructure using the command
```
./terraform apply
```
This will execute the deployment of the infrastructure and show the progress. For more information about the possible configurations, check the [TerraForm website](https://www.terraform.io/docs/configuration/syntax.html)

