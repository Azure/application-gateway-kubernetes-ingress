# Introduction

.. note::
    [Application Gateway for Containers](https://aka.ms/agc) has been released, which introduces numerous performance, resilience, and feature changes. Please consider leveraging Application Gateway for Containers for your next deployment.

The Application Gateway Ingress Controller allows [Azure Application Gateway](https://azure.microsoft.com/en-us/services/application-gateway/) to be used as the ingress for an [Azure Kubernetes Service](https://azure.microsoft.com/en-us/services/kubernetes-service/) aka AKS cluster.

As shown in the figure below, the ingress controller runs as a pod within the AKS cluster. It consumes [Kubernetes `Ingress` Resources](http://kubernetes.io/docs/user-guide/ingress/) and converts them to an Azure Application Gateway configuration which allows the gateway to load-balance traffic to Kubernetes pods.

![Azure Application Gateway + AKS](images/architecture.png)

## Reporting Issues

The best way to report an issue is to create a Github Issue for the project. Please include the following information when creating the issue:

* Subscription ID for AKS cluster.
* Subscription ID for Application Gateway.
* AKS cluster name/ARM Resource ID.
* Application Gateway name/ARM Resource ID.
* Ingress resource definition that might causing the problem.
* The Helm configuration used to install the ingress controller.
