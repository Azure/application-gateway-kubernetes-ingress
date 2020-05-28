// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import "time"

const (
	retryPause         = 10 * time.Second
	retryCount         = 3
	maxAuthRetryCount  = 10
	extendedRetryCount = 60
)

// RoleDefinition represents the role definition id
type RoleDefinition string

const (
	// These are role definitions that we use when using the role assignment client

	// Owner is the role defintion ID for the corresponding role in AAD
	Owner RoleDefinition = "8e3af657-a8ff-443c-a75c-2fe8c4bcb635"

	// Contributor is the role defintion ID for the corresponding role in AAD
	Contributor RoleDefinition = "b24988ac-6180-42a0-ab88-20f7382dd24c"

	// Reader is the role defintion ID for the corresponding role in AAD
	Reader RoleDefinition = "acdd72a7-3385-48ef-bd42-f606fba81ae7"
)

var (
	roleName = map[RoleDefinition]string{
		Contributor: "Contributor",
		Reader:      "Reader",
	}

	superSetRoles = map[RoleDefinition][]RoleDefinition{
		Contributor: {Contributor, Owner},
		Reader:      {Reader, Contributor, Owner},
	}
)
