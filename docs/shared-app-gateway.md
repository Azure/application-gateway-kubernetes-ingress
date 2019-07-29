## Shared App Gateway
By default AGIC assumes full ownership of the configuration of the App Gateway it is linked to. With minimal tweaks,
AGIC can share App Gateway with other Azure components. For instance, we could use the same App Gateway for our AKS
and also an app hosted on VMSS. We have the option to constrain AGIC, so it controls only a subset of the App
Gateway's properties.

### Example Scenario
Let's take an imaginary App Gatway, which manages traffic for 2 web sites:
  - prod.contoso.com
  - dev.contoso.com

We need to:
  - serve `dev.contoso.com` from a new AKS, using App Gateway and AGIC
  - serve  `prod.contoso.com` from an existing [Azure VMSS](https://azure.microsoft.com/en-us/services/virtual-machine-scale-sets/)

Until now (and by default) AGIC assumed 100% ownership of the App Gateway configuration. AGIC overwrites all of App
Gateway's configuration. If we were to manually create a listener for `prod.contoso.com` (on App Gateway), without
defining it in the Kubernetes Ingress, AGIC will delete the `prod.contoso.com` config within seconds.

To install AGIC and also serve `prod.contoso.com` from our VMSS machines, we must constrain AGIC to configuring 
`dev.contoso.com` only. This is is facilitated by instantiating the following
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

The command creates an `AzureIngressProhibitedTarget` object. This makes AGIC aware of the existence of
App Gateway config for `prod.contoso.com` and explicitly instructs it to avoid changing any configuration
related to that hostname.


### Enable Shared App Gateway for a new AGIC installation
To limit AGIC to a subset of the App Gateway configuration modify the `helm-config.yaml` template.
Under the `appgw:` section, add `shared` key and set it to to `true`.

```yaml
appgw:
    subscriptionId: <subscriptionId>    # existing field
    resourceGroup: <resourceGroupName>  # existing field
    name: <applicationGatewayName>      # existing field
    shared: true                        # <<<<< Add this field to enable shared App Gateway >>>>>
```

This instructs Helm to:
  - create a new CRD: `AzureIngressProhibitedTarget`
  - create new instance of `AzureIngressProhibitedTarget` called `prohibit-all-targets`

The default `prohibit-all-targets` prohibits AGIC from changing config for *any* host and path. Helm install
with `appgw.shared=true` will deploy AGIC, but will not make any changes to App Gateway.


### Broaden permissions
Enabling `appgw.shared=true` and `helm` installing AGIC, would result in the creation of a `prohibit-all-targets` object in your AKS.

View the contents of the object:
```bash
kubectl get AzureIngressProhibitedTarget prohibit-all-targets -o yaml
```

You can broaden the permissions AGIC has:
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

2. Only after you have created your own custom prohibition, you can delete the default one, which is too broad:

    ```bash
    kubectl delete AzureIngressProhibitedTarget prohibit-all-targets
    ```

### Enable Shared App Gateway for an existing AGIC installation
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