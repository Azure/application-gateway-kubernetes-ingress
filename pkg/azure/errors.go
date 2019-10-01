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

	// ErrNoSuchNamespace is an error message.
	ErrNoSuchNamespace = errors.New("namespace does not exist (AZUR002)")

	// ErrFailedGetToken is an error message.
	ErrFailedGetToken = errors.New("failed obtaining auth token (AZUR003)")

	// ErrGetArmAuth is an error message.
	ErrGetArmAuth = errors.New("failed arm auth (AZUR004)")

	// ErrAppGatewayNotFound is an error message.
	ErrAppGatewayNotFound = errors.New("not found (AZUR005)")
)
