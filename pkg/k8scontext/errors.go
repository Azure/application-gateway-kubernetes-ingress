// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import "errors"

var (
	// ErrorFetchingEnpdoints is an error.
	ErrorFetchingEnpdoints              = errors.New("FetchingEndpoints (KCTX001)")

	// ErrorUnknownSecretType is an error.
	ErrorUnknownSecretType              = errors.New("unknown secret type (KCTX002)")

	// ErrorCreatingFile is an error.
	ErrorCreatingFile                   = errors.New("unable to create temp file (KCTX003)")

	// ErrorMalformedSecret is an error.
	ErrorMalformedSecret                = errors.New("malformed secret (KCTX004)")

	// ErrorWritingToFile is an error.
	ErrorWritingToFile                  = errors.New("unable to write to file (KCTX005)")

	// ErrorExportingWithOpenSSL is an error.
	ErrorExportingWithOpenSSL           = errors.New("failed export with OpenSSL (KCTX006)")

	// ErrorInformersNotInitialized is an error.
	ErrorInformersNotInitialized        = errors.New("informers are not initialized (KCTX007)")

	// ErrorFailedInitialCacheSync is an error.
	ErrorFailedInitialCacheSync         = errors.New("failed initial sync of resources required for ingress (KCTX008)")

	// ErrorNoNodesFound is an error.
	ErrorNoNodesFound                   = errors.New("no nodes were found in the node list (KCTX009)")

	// ErrorUnrecognizedNodeProviderPrefix is an error.
	ErrorUnrecognizedNodeProviderPrefix = errors.New("providerID is not prefixed with azure:// (KCTX010)")

	// ErrorUnableToUpdateIngress is an error.
	ErrorUnableToUpdateIngress          = errors.New("ingress status update (KCTX011)")
)
