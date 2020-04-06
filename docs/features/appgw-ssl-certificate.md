### Prerequisites
This documents assumes you already have the following Azure tools and resources installed:
- [AKS](https://azure.microsoft.com/en-us/services/kubernetes-service/) with [Advanced Networking](https://docs.microsoft.com/en-us/azure/aks/configure-azure-cni) enabled
- [App Gateway v2](https://docs.microsoft.com/en-us/azure/application-gateway/create-zone-redundant) in the same virtual network as AKS
- [AAD Pod Identity](https://github.com/Azure/aad-pod-identity) installed on your AKS cluster
- [Cloud Shell](https://shell.azure.com/) is the Azure shell environment, which has `az` CLI, `kubectl`, and `helm` installed. These tools are required for the commands below.

Please use [Greenfeild Deployment](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/docs/setup/install-new.md) to install nonexistents.

## Create a certificate and configiure the certificate to AppGw
The certificate below should only be used for testing purpose.
```bash
appgwName=""
resgp=""

# generate certificate for testing
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -out azure-cli-app-tls.crt \
  -keyout azure-cli-app-tls.key \
  -subj "/CN=azure-cli-app"

openssl pkcs12 -export \
  -in azure-cli-tls.crt \
  -inkey sample-app-tls.key \
  -passout pass: -out azure-cli-cert.pfx

SecretValue=$(cat azure-cli-cert.pfx | base64)

# configure certificate to app gateway
az network application-gateway ssl-cert create \
  --resource-group $resgp \
  --gateway-name $appgwName \
  -n mysslcert \
  --data $SecretValue
```

## Configure certificate from Key Vault to AppGw
To configfure certificate from key vault to Application Gateway, an user-assigned managed identity will need to be created and assigned to AppGw, the managed identity will need to have GET secret access to KeyVault. 

```bash
appgwName=""
resgp=""
vaultName=""
location=""
agicIdentityPrincipalId=""

# One time operation, create Azure key vault and certificate (can done through portal as well)
az keyvault create -n $vaultName -g $resgp --enable-soft-delete -l $location

# One time operation, create user-assigned managed identity
az identity create -n appgw-id -g $resgp -l $location

# Assign the identity to Application Gateway
az network application-gateway identity assign --gateway-name $appgwName --resource-group $resgp --identity $identityID

# Assign the identity GET secret access to Azure Key Vault
identityID=$(az identity show -n appgw-id -g $resgp -o tsv --query "id")
identityPrincipal=$(az identity show -n appgw-id -g $resgp -o tsv --query "objectId")
az keyvault set-policy -n $vaultName -g $resgp --object-id $identityPrincipal  --secret-permissions get

# Create a cert on keyvault and add unversioned secret id to Application Gateway
az keyvault certificate create --vault-name $vaultName -n mycert -p "$(az keyvault certificate get-default-policy)"
versionedSecretId=$(az keyvault certificate show -n mycert --vault-name $vaultName --query "sid" -o tsv)
unversionedSecretId=$(echo $versionedSecretId | cut -d'/' -f-5) # remove the version from the url

az network application-gateway ssl-cert create -n mykvsslcert --gateway-name $appgwName --resource-group $resgp --key-vault-secret-id $unversionedSecretId # ssl certificate with name "mykvsslcert" will be configured on AppGw
```

## Testing the key vault certificate on Ingress
Since we have certificate from Key Vault configured in Application Gateway, we can then add the new annotation `appgw.ingress.kubernetes.io/appgw-ssl-certificate: mykvsslcert` in Kubernetes ingress to enable the feature.

```bash
# install an app
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: aspnetapp
  labels:
    app: aspnetapp
spec:
  containers:
  - image: "mcr.microsoft.com/dotnet/core/samples:aspnetapp"
    name: aspnetapp-image
    ports:
    - containerPort: 80
      protocol: TCP

---

apiVersion: v1
kind: Service
metadata:
  name: aspnetapp
spec:
  selector:
    app: aspnetapp
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80

---

apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: aspnetapp
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/appgw-ssl-certificate: mykvsslcert
spec:
  rules:
  - http:
      paths:
      - path: /
        backend:
          serviceName: aspnetapp
          servicePort: 80
EOF
```