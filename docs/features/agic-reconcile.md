## Reconcile scenario
When an Application Gateway is deployed through ARM template, a requirement is that the gateway configuration should contain a probe, listener, rule, backend pool and backend http setting. When such a template is re-deployed with minor changes (for example to WAF rules) on Gateway that is being controlled by AGIC, all the AGIC written rules are removed. Given such change on Application Gateway doesn’t trigger any events on AGIC, AGIC doesn’t reconcile the gateway back to the expected state. 

## Solution
To address the problem above, AGIC periodically checks if the latest gateway configuration is different from what it cached, and reconcile if needed to make gateway configuration is eventual correct.

## How to configure reconcile
There are two ways to configure AGIC reconcile via helm, and to use the new feature, make sure the AGIC version is at least at 1.2.0-rc1

### Configure inside helm values.yaml
`reconcilePeriodSeconds: 30`, it means AGIC checks the reconciling in every 30 seconds

### Configure from helm command line
Configure from helm install command(first time install) and helm upgrade command, helm version is v3
```bash
# helm fresh install
helm intall <releaseName> -f helm-config.yaml application-gateway-kubernetes-ingress/ingress-azure --version 1.2.0-rc1 --set reconcilePeriodSeconds=30 

# help upgrade
# --reuse-values, when upgrading, reuse the last release's values and merge in any overrides from the command line via --set and -f.
helm upgrade <releaseName> application-gateway-kubernetes-ingress/ingress-azure --reuse-values --version 1.2.0-rc1 --set reconcilePeriodSeconds=30
```