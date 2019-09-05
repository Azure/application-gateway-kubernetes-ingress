// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import "errors"

var (
	ErrorFetchingEnpdoints              = errors.New("FetchingEndpoints")
	ErrorUnknownSecretType              = errors.New("unknown secret type")
	ErrorCreatingFile                   = errors.New("unable to create temp file")
	ErrorMalformedSecret                = errors.New("malformed secret")
	ErrorWritingToFile                  = errors.New("unable to write to file")
	ErrorExportingWithOpenSSL           = errors.New("failed export with OpenSSL")
	ErrorInformersNotInitialized        = errors.New("informers are not initialized")
	ErrorFailedInitialCacheSync         = errors.New("failed initial sync of resources required for ingress")
	ErrorNoNodesFound                   = errors.New("no nodes were found in the node list")
	ErrorUnrecognizedNodeProviderPrefix = errors.New("providerID is not prefixed with azure://")
	ErrorUnableToUpdateIngress          = errors.New("ingress status update")
)
