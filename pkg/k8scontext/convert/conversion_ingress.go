package convert

import (
	multiClusterIngress "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/multiclusteringress/v1alpha1"
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

// FromMultiClusterIngress converts MultiClusterIngress CRD into neworking.k8s.io/v1/Ingress
func FromMultiClusterIngress(mci *multiClusterIngress.MultiClusterIngress) (*networkingv1.Ingress, bool) {
	if mci == nil {
		klog.Errorf("Unexpected, attempted converting nil MultiClusterIngresss to Ingress")
		return nil, false
	}
	v1Ing := &networkingv1.Ingress{}
	// remove last applied config, object model does not match
	for k := range mci.ObjectMeta.Annotations {
		if k == "kubectl.kubernetes.io/last-applied-configuration" {
			delete(mci.ObjectMeta.Annotations, k)
		}
	}
	mci.ObjectMeta.DeepCopyInto(&v1Ing.ObjectMeta)
	mci.Spec.Template.DeepCopyInto(&v1Ing.Spec)
	mci.Status.DeepCopyInto(&v1Ing.Status)

	v1Ing.APIVersion = networkingv1.SchemeGroupVersion.String()
	v1Ing.Kind = "Ingress"
	return v1Ing, true
}
