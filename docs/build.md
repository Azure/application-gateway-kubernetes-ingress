## Building

This is a CMake-based project. Build targets include:

- `ALL_BUILD` (default target) builds `appgw-ingress` and `dockerize` target
- `devenv` builds a docker image with configured development environment
- `vendor` installs dependency using `go mod` in a docker container with image from `devenv` target
- `appgw-ingress` builds the binary for this controller in a docker container with image from `devenv` target
- `dockerize` builds a docker image with the binary from `appgw-ingress` target
- `dockerpush` pushes the docker image to a container registry with prefix defined in CMake variable `<deployment_push_prefix>`

To run the CMake targets:

1. `mkdir build && cd build` creates and enters a build directory
2. `cmake ..` generates project configuration in the build directory
3. `cmake --build .` to build the default target,
    or `cmake --build . --target <target_name>` to specify a target to run from above


## Running it locally
This section outlines the environment variables and files necessary to successfully compile and run the Go binary, then connect it to an [Azure Kubernetes Service](https://docs.microsoft.com/en-us/azure/aks/intro-kubernetes).

### Obtain Azure Credentials

In order to run the Go binary locally and control a remote AKS server, you need Azure credentials. These will be stored in a JSON file in your home directory.

Follow [these instructions](https://docs.microsoft.com/en-us/dotnet/api/overview/azure/containerinstance?view=azure-dotnet#authentication) to create the `$HOME/.azure/azureAuth.json` file. The file is generated via:
```bash
az ad sp create-for-rbac --subscription <your-azure-subscription-id> --sdk-auth > $HOME/.azure/azureAuth.json
```
The file will contain a JSON blob with the following shape:
```json
{
  "clientId": "...",
  "clientSecret": "...",
  "subscriptionId": "<your-azure-resource-group>",
  "tenantId": "...",
  "activeDirectoryEndpointUrl": "https://login.microsoftonline.com",
  "resourceManagerEndpointUrl": "https://management.azure.com/",
  "activeDirectoryGraphResourceId": "https://graph.windows.net/",
  "sqlManagementEndpointUrl": "https://management.core.windows.net:8443/",
  "galleryEndpointUrl": "https://gallery.azure.com/",
  "managementEndpointUrl": "https://management.core.windows.net/"
}
```

### Startup Script
1. Create a `.dev` directory in your repo. This is already in `.gitignore` and will not be comitted.
1. Create an executable bash file in the root directory of this repo: `touch .dev/start.sh && chmod +x .dev/start.sh`
1. Add the following bash script to the `start.sh` file:
```bash
#!/bin/bash

set -aueo pipefail

export AZURE_AUTH_LOCATION=$HOME/.azure/azureAuth.json

export AKS_API="abc.westus2.azmk8s.io"  # YOUR AKS API Server

export APPGW_SUBSCRIPTION_ID=XYZ  # YOUR subscription ID
export APPGW_RESOURCE_GROUP=ABC  # YOUR resource group
export APPGW_NAME=123  # YOUR newly created Application Gateway's name

export KUBERNETES_WATCHNAMESPACE=default

# Build
GOOS=linux  # operating system target
GOBIN=`pwd`/bin

mkdir -p $GOBIN

 echo -e "\e[44;97m Compiling ... \e[0m"
if  go install -v ./cmd/appgw-ingress; then
    chmod -R 777 bin
    echo -e "\e[42;97m Build SUCCEEDED \e[0m"
else
    echo -e "\e[101;97m Build FAILED \e[0m"
    exit 1
fi

# Run
./bin/appgw-ingress \
    --in-cluster=false \
    --kubeconfig=$HOME/.kube/config \
    --apiserver-host=$AKS_API

```
Fill-in the values for your AKS cluster's **subscription**, **resource group**, **application gateway name**, and **AKS API server address**.
The script will create a `.build/` directory, compile and install the binary in it, then run the application.

### Run
With `$HOME/.azure/azureAuth.json` and `.dev/start.sh` created, you are ready to start the K8s ingress on your workstation. Execute `source .dev/start.sh` from within the root directory of the repo. The Ingress will connect to AKS, gather details on your running pods and configure the given Application Gateway.
