// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package sorter

import (
	"k8s.io/api/extensions/v1beta1"
)

// ByIngressUID is a facility to sort slices of Kubernetes Ingress by their UID
type ByIngressUID []*v1beta1.Ingress

func (a ByIngressUID) Len() int      { return len(a) }
func (a ByIngressUID) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByIngressUID) Less(i, j int) bool {
	return getIngressUID(a[i]) < getIngressUID(a[j])
}

func getIngressUID(ingress *v1beta1.Ingress) string {
	return string(ingress.UID)
}
