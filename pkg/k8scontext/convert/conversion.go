package convert

import (
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/klog/v2"
)

func fromExtensions(old *extensionsv1beta1.Ingress) (*networkingv1.Ingress, error) {
	v1Ing := &v1.Ingress{}
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
