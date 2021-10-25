// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
)

func serviceBackendPortToStr(port networkingv1.ServiceBackendPort) string {
	if port.Name != "" {
		return fmt.Sprintf(port.Name)
	}
	return fmt.Sprintf("%d", port.Number)
}
