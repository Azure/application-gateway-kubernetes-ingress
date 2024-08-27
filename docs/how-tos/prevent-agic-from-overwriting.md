# Preventing AGIC from removing certain rules

> **_NOTE:_** [Application Gateway for Containers](https://aka.ms/agc) has been released, which introduces numerous performance, resilience, and feature changes. Please consider leveraging Application Gateway for Containers for your next deployment.

> Note: This feature is **EXPERIMENTAL** with **limited support**. Use with caution.

By default AGIC assumes full ownership of the App Gateway it is linked to. AGIC version 0.8.0 and later allows
retaining rules to allow adding VMSS as backend along with AKS cluster.

Please **backup your App Gateway's configuration** before enabling this setting:

  1. using [Azure Portal](https://portal.azure.com/) navigate to your `App Gateway` instance
  2. from `Export template` click `Download`

The zip file you downloaded will have JSON templates, bash, and PowerShell scripts you could use to restore App Gateway

## Example Scenario

Let's look at an imaginary App Gateway, which manages traffic for 2 web sites:

- `dev.contoso.com` - hosted on a new AKS, using App Gateway and AGIC
- `prod.contoso.com` - hosted on an [Azure VMSS](https://azure.microsoft.com/en-us/services/virtual-machine-scale-sets/)

With default settings, AGIC assumes 100% ownership of the App Gateway it is pointed to. AGIC overwrites all of App
Gateway's configuration. If we were to manually create a listener for `prod.contoso.com` (on App Gateway), without
defining it in the Kubernetes Ingress, AGIC will delete the `prod.contoso.com` config within seconds.

To install AGIC and also serve `prod.contoso.com` from our VMSS machines, we must constrain AGIC to configuring
`dev.contoso.com` only. This is facilitated by instantiating the following
[CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/):

```bash
cat <<EOF | kubectl apply -f -
apiVersion: "appgw.ingress.k8s.io/v1"
kind: AzureIngressProhibitedTarget
metadata:
  name: prod-contoso-com
spec:
  hostname: prod.contoso.com
EOF
```

The command above creates an `AzureIngressProhibitedTarget` object. This makes AGIC (version 0.8.0 and later) aware of the existence of
App Gateway config for `prod.contoso.com` and explicitly instructs it to avoid changing any configuration
related to that hostname.

## Enable with new AGIC installation

To limit AGIC (version 0.8.0 and later) to a subset of the App Gateway configuration modify the `helm-config.yaml` template.
Under the `appgw:` section, add `shared` key and set it to to `true`.

```yaml
appgw:
    subscriptionId: <subscriptionId>    # existing field
    resourceGroup: <resourceGroupName>  # existing field
    name: <applicationGatewayName>      # existing field
    shared: true                        # <<<<< Add this field to enable shared App Gateway >>>>>
```

Apply the Helm changes:

  1. Ensure the `AzureIngressProhibitedTarget` CRD is installed with:

      ```bash
      kubectl apply -f https://raw.githubusercontent.com/Azure/application-gateway-kubernetes-ingress/ae695ef9bd05c8b708cedf6ff545595d0b7022dc/crds/AzureIngressProhibitedTarget.yaml
      ```

  2. Update Helm:

      ```bash
      helm upgrade \
          --recreate-pods \
          -f helm-config.yaml \
          ingress-azure application-gateway-kubernetes-ingress/ingress-azure
      ```

As a result your AKS will have a new instance of `AzureIngressProhibitedTarget` called `prohibit-all-targets`:

```bash
kubectl get AzureIngressProhibitedTargets prohibit-all-targets -o yaml
```

The object `prohibit-all-targets`, as the name implies, prohibits AGIC from changing config for *any* host and path.
Helm install with `appgw.shared=true` will deploy AGIC, but will not make any changes to App Gateway.

## Broaden permissions

Since Helm with `appgw.shared=true` and the default `prohibit-all-targets` blocks AGIC from applying any config.

Broaden AGIC permissions with:

1. Create a new `AzureIngressProhibitedTarget` with your specific setup:

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: "appgw.ingress.k8s.io/v1"
    kind: AzureIngressProhibitedTarget
    metadata:
      name: your-custom-prohibitions
    spec:
      hostname: your.own-hostname.com
    EOF
    ```

**NOTE:** To prohibit AGIC from making changes, in addition to *hostname*, a list of URL paths can also be configured as part of your prohibited policy, please refer to the [schema](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/crds/AzureIngressProhibitedTarget-v1-CRD-v1.yaml) for details.

2. Only after you have created your own custom prohibition, you can delete the default one, which is too broad:

    ```bash
    kubectl delete AzureIngressProhibitedTarget prohibit-all-targets
    ```

## Enable for an existing AGIC installation

Let's assume that we already have a working AKS, App Gateway, and configured AGIC in our cluster. We have an Ingress for
`prod.contosor.com` and are successfully serving traffic for it from AKS. We want to add `staging.contoso.com` to our
existing App Gateway, but need to host it on a [VM](https://azure.microsoft.com/en-us/services/virtual-machines/). We
are going to re-use the existing App Gateway and manually configure a listener and backend pools for
`staging.contoso.com`. But manually tweaking App Gateway config (via
[portal](https://portal.azure.com), [ARM APIs](https://docs.microsoft.com/en-us/rest/api/resources/) or
[Terraform](https://www.terraform.io/)) would conflict with AGIC's assumptions of full ownership. Shortly after we apply
changes, AGIC will overwrite or delete them.

We can prohibit AGIC from making changes to a subset of configuration.

1. Create an `AzureIngressProhibitedTarget` object:

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: "appgw.ingress.k8s.io/v1"
    kind: AzureIngressProhibitedTarget
    metadata:
      name: manually-configured-staging-environment
    spec:
      hostname: staging.contoso.com
    EOF
    ```

2. View the newly created object:

    ```bash
    kubectl get AzureIngressProhibitedTargets
    ```

3. Modify App Gateway config via portal - add listeners, routing rules, backends etc. The new object we created
(`manually-configured-staging-environment`) will prohibit AGIC from overwriting App Gateway configuration related to
`staging.contoso.com`.
