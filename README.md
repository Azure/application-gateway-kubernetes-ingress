# Azure Application Gateway Ingress Controller
The Application Gateway Ingress Controller allows the Azure Application Gateway to be used as the ingress for an AKS (Azure Kubernetes Service) cluster. As shown in the figure below, the ingress controller runs as a pod within the AKS cluster. It consumes [Kubernetes Ingress Resources](https://kubernetes.io/docs/concepts/services-networking/ingress/) and converts them to an Azure Application Gateway configuration which allows the gateway to load-balance traffic to Kubernetes pods.
![Azure Applicatieon Gateway + AKS](docs/images/architecture.png)

## Setup and Usage
Refer to the [Documentation](docs/install.md) to find instructions on installing the ingress controller on an AKS cluster, as well as tutorials on using it to configure an existing Azure Applicaton Gateway to act as an ingress to AKS.

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
