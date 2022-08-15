module github.com/Azure/application-gateway-kubernetes-ingress/scripts/e2e

go 1.16

require (
	github.com/Azure/application-gateway-kubernetes-ingress v0.0.0
	github.com/Azure/azure-sdk-for-go v66.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.28
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.11
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/emicklei/go-restful v2.16.0+incompatible // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/google/uuid v1.3.0
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.20.0
	k8s.io/api v0.24.3
	k8s.io/apimachinery v0.24.3
	k8s.io/client-go v0.24.3
	k8s.io/klog/v2 v2.70.1
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
)

replace github.com/Azure/application-gateway-kubernetes-ingress => ../..
