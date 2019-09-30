// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"errors"
)

var (
	// ErrNoSuchNamespace is an error message.
	ErrNoSuchNamespace = errors.New("namespace does not exist (MAIN001)")

	// ErrFailedGetToken is an error message.
	ErrFailedGetToken = errors.New("failed obtaining auth token (MAIN002)")

	// ErrGetArmAuth is an error message.
	ErrGetArmAuth = errors.New("failed arm auth (MAIN003)")

	// ErrAppGatewayNotFound is an error message.
	ErrAppGatewayNotFound = errors.New("not found (MAIN004)")
)
