![Ingress Controller - Status](https://img.shields.io/badge/project--status-beta-orange.svg)
[![Build Status](https://travis-ci.org/Azure/application-gateway-kubernetes-ingress.svg?branch=master)](https://travis-ci.org/Azure/application-gateway-kubernetes-ingress)

The Application Gateway Ingress Controller allows the [Azure Application Gateway](https://azure.microsoft.com/en-us/services/application-gateway/) to be used as the ingress for an [Azure Kubernetes Service](https://azure.microsoft.com/en-us/services/kubernetes-service/) aka AKS cluster. As shown in the figure below, the ingress controller runs as a pod within the AKS cluster. It consumes [Kubernetes Ingress Resources](https://kubernetes.io/docs/concepts/services-networking/ingress/) and converts them to an Azure Application Gateway configuration which allows the gateway to load-balance traffic to Kubernetes pods.

![Azure Application Gateway + AKS](docs/images/architecture.png)

## Setup
* **Greenfield Deployment**: If you are starting from scratch, refer to these [installation](docs/install-new.md) instructions which outlines steps to deploy an AKS cluster with Application Gateway and install application gateway ingress controller on the AKS cluster.
* **Brownfield Deployment**: If you have an existing AKS cluster and Application Gateway, refer to these [installation](docs/install-existing.md) instructions to install application gateway ingress controller on the AKS cluster.

## Usage
Refer to the [tutorials](docs/tutorial.md) to understand how you can expose an AKS service over HTTP or HTTPS, to the internet, using an Azure Application Gateway.

## Annotations
The Kubernetes Ingress specification does not allow all features of Application Gateway to be exposed through the ingress resource. Therefore we have introduced application gateway ingress controller specific annotations to expose application gateway features through an ingress resource. Please refer to the [annotations](docs/annotations.md) to understand the various annotations supported by the ingress controller, and the corresponding features that can be turned on in the application gateway for a given annotation.

## Reporting Issues
The best way to report an issue is to create a Github Issue for the project. Please include the following information when creating the issue:
* Subscription ID for AKS cluster.
* Subscription ID for Application Gateway.
* AKS cluster name/ARM Resource ID.
* Application Gateway name/ARM Resource ID.
* Ingress resource definition that might causing the problem.
* The Helm configuration used to install the ingress controller.

## Contributing
This is a Golang project. You can find the build instructions of the project in the [Developer Guide](docs/build.md).

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
