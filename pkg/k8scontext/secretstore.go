// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"bytes"
	"os"
	"os/exec"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
)

const (
	recognizedSecretType = "kubernetes.io/tls"
	tlsKey               = "tls.key"
	tlsCrt               = "tls.crt"
)

// SecretsKeeper is the interface definition for secret store
type SecretsKeeper interface {
	GetPfxCertificate(secretKey string) []byte
	ConvertSecret(secretKey string, secret *v1.Secret) error
	delete(secretKey string)
}

// SecretsStore maintains a cache of the deployment secrets.
type SecretsStore struct {
	conversionSync sync.Mutex
	Cache          cache.ThreadSafeStore
}

// NewSecretStore creates a new SecretsKeeper object
func NewSecretStore() SecretsKeeper {
	return &SecretsStore{
		Cache: cache.NewThreadSafeStore(cache.Indexers{}, cache.Indices{}),
	}
}

// GetPfxCertificate returns the certificate for the given secret key.
func (s *SecretsStore) GetPfxCertificate(secretKey string) []byte {
	if certInterface, exists := s.Cache.Get(secretKey); exists {
		if cert, ok := certInterface.([]byte); ok {
			return cert
		}
	}
	return nil
}

func (s *SecretsStore) delete(secretKey string) {
	s.conversionSync.Lock()
	defer s.conversionSync.Unlock()

	s.Cache.Delete(secretKey)
}

// ConvertSecret converts a secret to a PKCS12.
func (s *SecretsStore) ConvertSecret(secretKey string, secret *v1.Secret) error {
	s.conversionSync.Lock()
	defer s.conversionSync.Unlock()

	// check if this is a secret with the correct type
	if secret.Type != recognizedSecretType {
		return controllererrors.NewErrorf(
			controllererrors.ErrorUnknownSecretType,
			"secret [%v] is not type kubernetes.io/tls", secretKey,
		)
	}

	if len(secret.Data[tlsKey]) == 0 || len(secret.Data[tlsCrt]) == 0 {
		return controllererrors.NewErrorf(
			controllererrors.ErrorMalformedSecret,
			"secret [%v] is malformed, tls.key or tls.crt is not defined", secretKey,
		)
	}

	tempfileCert, err := os.CreateTemp("", "appgw-ingress-cert")
	if err != nil {
		return controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorCreatingFile,
			err,
			"unable to create temporary file for certificate conversion",
		)
	}
	defer os.Remove(tempfileCert.Name())

	tempfileKey, err := os.CreateTemp("", "appgw-ingress-key")
	if err != nil {
		return controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorCreatingFile,
			err,
			"unable to create temporary file for certificate conversion",
		)
	}
	defer os.Remove(tempfileKey.Name())

	if err := writeFileDecode(secret.Data["tls.crt"], tempfileCert); err != nil {
		return controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorWritingToFile,
			err,
			"unable to write secret [%v].tls.crt to temporary file", secretKey,
		)
	}

	if err := writeFileDecode(secret.Data["tls.key"], tempfileKey); err != nil {
		return controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorWritingToFile,
			err,
			"unable to write secret [%v].tls.key to temporary file", secretKey,
		)
	}

	// both cert and key are in temp file now, call openssl
	var cout, cerr bytes.Buffer
	cmd := exec.Command("openssl", "pkcs12", "-export", "-in", tempfileCert.Name(), "-inkey", tempfileKey.Name(), "-password", "pass:msazure")
	cmd.Stderr = &cerr
	cmd.Stdout = &cout

	// if openssl exited with an error or the output is empty, report error
	if err := cmd.Run(); err != nil || len(cout.Bytes()) == 0 {
		return controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorExportingWithOpenSSL,
			err,
			"unable to export using openssl, error=[%v], stderr=[%v]", err, cerr.String(),
		)
	}

	pfxCert := cout.Bytes()

	// TODO i'm not sure if comparison against existing certificate can help
	// us optimize by eliminating some events
	_, exists := s.Cache.Get(secretKey)
	if exists {
		s.Cache.Update(secretKey, pfxCert)
	} else {
		s.Cache.Add(secretKey, pfxCert)
	}

	return nil
}

func writeFileDecode(data []byte, fileHandle *os.File) error {
	if _, err := fileHandle.Write(data); err != nil {
		return err
	}
	if err := fileHandle.Close(); err != nil {
		return err
	}
	return nil
}
