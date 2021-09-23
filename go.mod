module github.com/Azure/application-gateway-kubernetes-ingress

go 1.16

require (
	github.com/Azure/azure-sdk-for-go v57.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.19
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.8
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/deckarep/golang-set v1.7.1
	github.com/getlantern/deepcopy v0.0.0-20160317154340-7f45deb8130a
	github.com/knative/pkg v0.0.0-20190619032946-d90a9bc97dde
	github.com/kylelemons/godebug v1.1.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.14.0
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/pflag v1.0.5
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	k8s.io/klog/v2 v2.9.0
)
