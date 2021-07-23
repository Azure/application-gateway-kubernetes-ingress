package convert

import (
	multiClusterIngress "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azuremulticlusteringress/v1alpha1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/klog/v2"
)

func fromExtensions(old *extensionsv1beta1.Ingress) (*networkingv1.Ingress, error) {
	v1Ing := &networkingv1.Ingress{}
	err := Convert_v1beta1_Ingress_To_networking_Ingress(old, v1Ing, nil)
	v1Ing.APIVersion = networkingv1.SchemeGroupVersion.String()
	v1Ing.Kind = "Ingress"
	return v1Ing, err
}

// ToIngressV1 converts to V1 ingress
func ToIngressV1(obj interface{}) (*networkingv1.Ingress, bool) {
	oldVersion, inExtension := obj.(*extensionsv1beta1.Ingress)
	if inExtension {
		ing, err := fromExtensions(oldVersion)
		if err != nil {
			klog.Errorf("unexpected error converting Ingress from extensions package: %v", err)
			return nil, false
		}

		return ing, true
	}

	if ing, ok := obj.(*networkingv1.Ingress); ok {
		return ing, true
	}

	return nil, false
}

func FromMultiClusterIngress(mci *multiClusterIngress.AzureMultiClusterIngress) (*networkingv1.Ingress, bool) {
	if mci == nil {
		klog.Errorf("Unexpected, attempted converting nil MultiClusterIngresss to Ingress")
		return nil, false
	}
	v1Ing := &networkingv1.Ingress{}
	v1Ing.ObjectMeta = mci.ObjectMeta
	v1Ing.Spec = mci.Spec
	v1Ing.APIVersion = networkingv1.SchemeGroupVersion.String()
	v1Ing.Kind = "Ingress"
	return v1Ing, true
}
