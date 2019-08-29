// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import "errors"

var (
	ErrorFetchingEnpdoints    = errors.New("FetchingEndpoints")
	ErrorUnknownSecretType    = errors.New("unknown secret type")
	ErrorCreatingFile         = errors.New("unable to create temp file")
	ErrorMalformedSecret      = errors.New("malformed secret")
	ErrorWritingToFile        = errors.New("unable to write to file")
	ErrorExportingWithOpenSSL = errors.New("failed export with OpenSSL")
)
