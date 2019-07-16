// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
)

type listenersByName map[listenerName]n.ApplicationGatewayHTTPListener

// getListenersByName indexes listeners by their name
func (er *ExistingResources) getListenersByName() listenersByName {
	if er.listenersByName != nil {
		return er.listenersByName
	}
	listenersByName := make(listenersByName)
	for _, listener := range er.Listeners {
		listenersByName[listenerName(listenerName(*listener.Name))] = listener
	}
	er.listenersByName = listenersByName
	return listenersByName
}
