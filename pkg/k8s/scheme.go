package k8s

import (
	nodenetworkconfig_v1alpha "github.com/Azure/azure-container-networking/crd/nodenetworkconfig/api/v1alpha"
	overlayextensionconfig_v1alpha1 "github.com/Azure/azure-container-networking/crd/overlayextensionconfig/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewScheme builds and returns k8s schemes used by ALB Controller.
func NewScheme() (*runtime.Scheme, error) {
	s := runtime.NewScheme()
	sb := runtime.SchemeBuilder{
		// Azure CNI CRDs
		overlayextensionconfig_v1alpha1.AddToScheme,
		nodenetworkconfig_v1alpha.AddToScheme,
	}

	if err := sb.AddToScheme(s); err != nil {
		return nil, err
	}

	return s, nil
}
