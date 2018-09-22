// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"strings"

	"k8s.io/api/core/v1"
)

func printEndpoints(endpoints v1.Endpoints) {
	fmt.Printf("Endpoint [%s]\n", endpoints.Name)
	for _, subset := range endpoints.Subsets {
		ports := subset.Ports
		tmp := make([]string, 0)
		for _, port := range ports {
			tmp = append(tmp, port.Name)
		}
		portsString := strings.Join(tmp, ",")
		fmt.Printf(" - subset ports=[%s]\n", portsString)
	}
}
