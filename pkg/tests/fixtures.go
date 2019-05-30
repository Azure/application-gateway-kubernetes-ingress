// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package tests

import (
	"fmt"
	"io/ioutil"

	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
)

// GetIngress creates an Ingress test fixture.
func GetIngress() (*v1beta1.Ingress, error) {
	return getIngress("ingress.yaml")
}

// GetIngressComplex creates an Ingress test fixture with multiple backends and path rules.
func GetIngressComplex() (*v1beta1.Ingress, error) {
	return getIngress("ingress-complex.yaml")
}

// GetIngressNamespaced creates 2 Ingress test fixtures in different namespaces.
func GetIngressNamespaced() (*[]v1beta1.Ingress, error) {
	ingr1, err := getIngress("ingress-namespace-1.yaml")
	if err != nil {
		glog.Fatal(err)
	}
	ingr2, err := getIngress("ingress-namespace-2.yaml")
	if err != nil {
		glog.Fatal(err)
	}
	return &[]v1beta1.Ingress{*ingr1, *ingr2}, nil
}

func getIngress(fileName string) (*v1beta1.Ingress, error) {
	ingr, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Print(err)
	}

	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(ingr, nil, nil)
	if err != nil {
		return nil, err
	}
	return obj.(*v1beta1.Ingress), nil
}
