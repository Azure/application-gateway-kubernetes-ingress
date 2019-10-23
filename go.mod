module github.com/Azure/application-gateway-kubernetes-ingress

go 1.12

require (
	contrib.go.opencensus.io/exporter/ocagent v0.5.0 // indirect
	github.com/Azure/azure-sdk-for-go v32.5.0+incompatible
	github.com/Azure/go-autorest v12.3.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.9.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/axw/gocov v1.0.0 // indirect
	github.com/deckarep/golang-set v1.7.1
	github.com/eapache/channels v1.1.0
	github.com/eapache/queue v1.1.0 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/getlantern/deepcopy v0.0.0-20160317154340-7f45deb8130a
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.9.1 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/knative/pkg v0.0.0-20190619032946-d90a9bc97dde
	github.com/matm/gocov-html v0.0.0-20160206185555-f6dd0fd0ebc7 // indirect
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.1.0
	github.com/spf13/pflag v1.0.3
	go.opencensus.io v0.22.0 // indirect
	golang.org/x/crypto v0.0.0-20190611184440-5c40567a22f8 // indirect
	golang.org/x/lint v0.0.0-20190909230951-414d861bb4ac // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/api v0.7.0 // indirect
	google.golang.org/appengine v1.6.1 // indirect
	google.golang.org/genproto v0.0.0-20190817000702-55e96fffbd48 // indirect
	google.golang.org/grpc v1.21.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/api v0.0.0-20190614205929-e4e27c96b39a
	k8s.io/apimachinery v0.0.0-20190612125636-6a5db36e93ad
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v0.3.3 // indirect
	k8s.io/kube-openapi v0.0.0-20190603182131-db7b694dc208 // indirect
	k8s.io/utils v0.0.0-20190607212802-c55fbcfc754a // indirect
)

replace (
	golang.org/x/sync => golang.org/x/sync v0.0.0-20181108010431-42b317875d0f
	golang.org/x/sys => golang.org/x/sys v0.0.0-20190209173611-3b5209105503
	golang.org/x/tools => golang.org/x/tools v0.0.0-20190313210603-aa82965741a9
	k8s.io/api => k8s.io/api v0.0.0-20190612125737-db0771252981
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190612125636-6a5db36e93ad
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190612125919-5c45477a8ae7
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190612125529-c522cb6c26aa
	k8s.io/component-base => k8s.io/component-base v0.0.0-20190612130303-4062e14deebe
)

replace github.com/golang/lint v0.0.0-20190409202823-959b441ac422 => golang.org/x/lint v0.0.0-20190409202823-959b441ac422
