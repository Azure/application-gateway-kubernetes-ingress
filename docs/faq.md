# Frequrently Asked Questions: [WIP]

> **_NOTE:_** [Application Gateway for Containers](https://aka.ms/agc) has been released, which introduces numerous performance, resilience, and feature changes. Please consider leveraging Application Gateway for Containers for your next deployment.

* [What is an Ingress Controller](#what-is-an-ingress-controller)
* [Can single ingress controller instance manage multiple Application Gateway](#can-single-ingress-controller-instance-manage-multiple-application-gateway)

## What is an Ingress Controller

Kubernetes allows creation of `deployment` and `service` resource to expose a group of pods internally in the cluster. To expose the same service externally, an [`Ingress`](https://kubernetes.io/docs/concepts/services-networking/ingress/) resource is defined which provides load balancing, SSL termination and name-based virtual hosting.
To satify this `Ingress` resource, an Ingress Controller is required which listens for any changes to `Ingress` resources and configures the load balancer policies.

The Application Gateway Ingress Controller allows [Azure Application Gateway](https://azure.microsoft.com/en-us/services/application-gateway/) to be used as the ingress for an [Azure Kubernetes Service](https://azure.microsoft.com/en-us/services/kubernetes-service/) aka AKS cluster.

## Can single ingress controller instance manage multiple Application Gateway

Currently, One instance of Ingress Controller can only be associated to one Application Gateway.