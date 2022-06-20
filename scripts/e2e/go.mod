module github.com/Azure/application-gateway-kubernetes-ingress/scripts/e2e

go 1.16

require (
	github.com/Azure/application-gateway-kubernetes-ingress v0.0.0
	github.com/Azure/azure-sdk-for-go v57.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.19
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.8
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/google/uuid v1.2.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.19.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	k8s.io/klog/v2 v2.9.0
	sigs.k8s.io/structured-merge-diff/v4 v4.1.0 // indirect
)

replace github.com/Azure/application-gateway-kubernetes-ingress => ../..
