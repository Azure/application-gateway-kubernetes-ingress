// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package sorter

import (
	"fmt"

	networking "k8s.io/api/networking/v1beta1"
)

// ByIngressName is a facility to sort slices of Kubernetes Ingress by their UID
type ByIngressName []*networking.Ingress

func (a ByIngressName) Len() int      { return len(a) }
func (a ByIngressName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByIngressName) Less(i, j int) bool {
	return getIngressName(a[i]) < getIngressName(a[j])
}

func getIngressName(ingress *networking.Ingress) string {
	return fmt.Sprintf("%s/%s", ingress.Namespace, ingress.Name)
}
