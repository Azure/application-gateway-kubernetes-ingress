# Helm Values Configuration Options

> **_NOTE:_** [Application Gateway for Containers](https://aka.ms/agc) has been released, which introduces numerous performance, resilience, and feature changes. Please consider leveraging Application Gateway for Containers for your next deployment.

## Available options

| Field | Default | Description |
| - | - | - |
| `verbosityLevel`| 3 | Sets the verbosity level of the AGIC logging infrastructure. See [Logging Levels](logging-levels.md) for possible values. |
| `reconcilePeriodSeconds` | | Enable periodic reconciliation to checks if the latest gateway configuration is different from what it cached. Range: 30 - 300 seconds. Disabled by default. |
| `appgw.applicationGatewayID` | | Resource Id of the Application Gateway. Example: `applicationgatewayd0f0` |
| `appgw.subscriptionId` | Default is agent node pool's subscriptionId derived from CloudProvider config  | The Azure Subscription ID in which App Gateway resides. Example: `a123b234-a3b4-557d-b2df-a0bc12de1234` |
| `appgw.resourceGroup` | Default is agent node pool's resource group derived from CloudProvider config | Name of the Azure Resource Group in which App Gateway was created. Example: `app-gw-resource-group` |
| `appgw.name` | | Name of the Application Gateway. Example: `applicationgatewayd0f0` |
| `appgw.environment`| `AZUREPUBLICCLOUD` | Specify which cloud environment. Possbile values: `AZURECHINACLOUD`, `AZUREGERMANCLOUD`, `AZUREPUBLICCLOUD`, `AZUREUSGOVERNMENTCLOUD` |
| `appgw.shared` | false | This boolean flag should be defaulted to `false`. Set to `true` should you need a [Shared App Gateway](how-tos/prevent-agic-from-overwriting.md). |
| `appgw.subResourceNamePrefix` | No prefix if empty | Prefix that should be used in the naming of the Application Gateway's sub-resources|
| `kubernetes.watchNamespace` | Watches all if empty | Specify the name space, which AGIC should watch. This could be a single string value, or a comma-separated list of namespaces. |
| `kubernetes.securityContext` | `runAsUser: 0` | Specify the pod security context to use with AGIC deployment. By default, AGIC will assume `root` permission. Jump to [Run without root](#run-without-root) for more information. |
| `kubernetes.containerSecurityContext` | `{}` | Specify the [container security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-container) to use with AGIC deployment. |
| `kubernetes.podAnnotations` | `{}` | Specify custom annotations for AGIC pod |
| `kubernetes.resources` | `{}` | Specify [resource quota](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/) for AGIC pod |
| `kubernetes.nodeSelector` | `{}` | Scheduling node selector |
| `kubernetes.tolerations` | `[]` | Scheduling tolerations |
| `kubernetes.affinity` | `{}` | Scheduling affinity |
| `kubernetes.volumes.extraVolumes` | `{}` | Specify additional volumes for the AGIC pod. This can be useful when [running on a `readOnlyRootFilesystem`](#run-with-read-only-root-filesystem), as AGIC requires a writeable `/tmp` directory. |
| `kubernetes.volumes.extraVolumeMounts` | `{}` | Specify additional volume mounts for the AGIC pod. This can be useful when [running on a `readOnlyRootFilesystem`](#run-with-read-only-root-filesystem), as AGIC requires a writeable `/tmp` directory. |
| `kubernetes.ingressClass` | `azure/application-gateway` | Specify a [custom ingress class](features/custom-ingress-class.md) which will be used to match `kubernetes.io/ingress.class` in ingress manifest |
| `rbac.enabled` | false | Specify true if kubernetes cluster is rbac enabled |
| `armAuth.type` | | could be `aadPodIdentity` or `servicePrincipal` |
| `armAuth.identityResourceID` | | Resource ID of the Azure Managed Identity |
| `armAuth.identityClientId` | | The Client ID of the Identity. See below for more information on Identity |
| `armAuth.secretJSON` | | Only needed when Service Principal Secret type is chosen (when `armAuth.type` has been set to `servicePrincipal`) |
| `nodeSelector` | `{}` | (Legacy: use `kubernetes.nodeSelector` instead) Scheduling node selector |

## Example

```yaml
appgw:
    applicationGatewayID: <application-gateway-resource-id>
    environment: "AZUREUSGOVERNMENTCLOUD" # default: AZUREPUBLICCLOUD

armAuth:
    type: aadPodIdentity
    identityResourceID: <identityResourceId>
    identityClientID:  <identityClientId>

kubernetes:
  nodeSelector: {}
  tolerations: []
  affinity: {}

rbac:
    enabled: false
```

### Run without root

By default, AGIC will assume `root` permission which allows it to read `cloud-provider` config and get meta-data information about the cluster.
If you want AGIC to run without `root` access, then make sure that AGIC is installed with at least the following information to run successfully:

```yaml
appgw:
    applicationGatewayID: <application-gateway-resource-id>
    # OR
    subscriptionId: <subscription-id>
    resourceGroup: <resource-group-name>
    name: <application-gateway-name>

kubernetes:
    securityContext:
        runAsUser: 1000 # appgw-ingress-user
```

> **Note:** AGIC also uses `cloud-provider` config to get Node's Virtual Network Name / Subscription and Route table name. If AGIC is not able to reach this information,  It will skip assigning the Node's route table to Application Gateway's subnet which is required when using `kubenet` network plugin. To workaround, this assignment can be performed manually.

### Run with read-only root filesystem

To run AGIC with `readOnlyRootFilesystem`, the following additional configuration items are required:

```yaml
kubernetes:
    containerSecurityContext:
        readOnlyRootFilesystem: true
    volumes:
        extraVolumes:
        - name: tmp
          emptyDir: {}
        extraVolumeMounts:
        - name: tmp
          mountPath: /tmp
```

> **Note:** AGIC needs to be able to write to the `/tmp` directory.
