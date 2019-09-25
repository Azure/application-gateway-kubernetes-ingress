// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import (
	"errors"
)

var (
	// ErrMissingResourceGroup is an error message.
	ErrMissingResourceGroup = errors.New("unable to locate AKS resource group (AZUR001)")
)
