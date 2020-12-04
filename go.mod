module github.com/Azure/application-gateway-kubernetes-ingress

go 1.13

require (
	github.com/Azure/azure-sdk-for-go v43.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.6
	github.com/Azure/go-autorest/autorest/azure/auth v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/deckarep/golang-set v1.7.1
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/getlantern/deepcopy v0.0.0-20160317154340-7f45deb8130a
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/knative/pkg v0.0.0-20190619032946-d90a9bc97dde
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.7.0
	github.com/pkg/errors v0.8.1 // indirect
	github.com/prometheus/client_golang v1.1.0
	github.com/spf13/pflag v1.0.5
	google.golang.org/appengine v1.6.1 // indirect
	k8s.io/api v0.0.0-20200326015715-b5bd82427fa8
	k8s.io/apimachinery v0.0.0-20200326015016-e92250ad09d8
	k8s.io/client-go v0.16.7
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.4.0
)

replace (
	golang.org/x/sync => golang.org/x/sync v0.0.0-20181108010431-42b317875d0f
	golang.org/x/sys => golang.org/x/sys v0.0.0-20190209173611-3b5209105503
	golang.org/x/tools => golang.org/x/tools v0.0.0-20190313210603-aa82965741a9
	k8s.io/api => k8s.io/api v0.0.0-20200326015715-b5bd82427fa8
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20200326015016-e92250ad09d8
	k8s.io/client-go => k8s.io/client-go v0.0.0-20200326020446-6240434e1ad6
	k8s.io/component-base => k8s.io/component-base v0.0.0-20190612130303-4062e14deebe
)

replace github.com/golang/lint v0.0.0-20190409202823-959b441ac422 => golang.org/x/lint v0.0.0-20190409202823-959b441ac422
