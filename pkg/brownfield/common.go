// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"

func portFromListener(listener *network.ApplicationGatewayHTTPListener) int32 {
	if listener != nil && listener.Protocol == network.HTTPS {
		return int32(443)
	}
	return int32(80)
}
