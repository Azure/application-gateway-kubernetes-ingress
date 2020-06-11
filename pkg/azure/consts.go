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
