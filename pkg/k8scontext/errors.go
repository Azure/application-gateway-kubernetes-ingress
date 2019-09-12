// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import "errors"

var (
	// ErrorFetchingEnpdoints is an error.
	ErrorFetchingEnpdoints              = errors.New("FetchingEndpoints")

	// ErrorUnknownSecretType is an error.
	ErrorUnknownSecretType              = errors.New("unknown secret type")

	// ErrorCreatingFile is an error.
	ErrorCreatingFile                   = errors.New("unable to create temp file")

	// ErrorMalformedSecret is an error.
	ErrorMalformedSecret                = errors.New("malformed secret")

	// ErrorWritingToFile is an error.
	ErrorWritingToFile                  = errors.New("unable to write to file")

	// ErrorExportingWithOpenSSL is an error.
	ErrorExportingWithOpenSSL           = errors.New("failed export with OpenSSL")

	// ErrorInformersNotInitialized is an error.
	ErrorInformersNotInitialized        = errors.New("informers are not initialized")

	// ErrorFailedInitialCacheSync is an error.
	ErrorFailedInitialCacheSync         = errors.New("failed initial sync of resources required for ingress")

	// ErrorNoNodesFound is an error.
	ErrorNoNodesFound                   = errors.New("no nodes were found in the node list")

	// ErrorUnrecognizedNodeProviderPrefix is an error.
	ErrorUnrecognizedNodeProviderPrefix = errors.New("providerID is not prefixed with azure://")

	// ErrorUnableToUpdateIngress is an error.
	ErrorUnableToUpdateIngress          = errors.New("ingress status update")
)
