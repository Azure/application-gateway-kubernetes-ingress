// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"bytes"

	"io/ioutil"
	"os"
	"os/exec"
	"sync"

	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

// SecretStore is the interface definition for secret store
type SecretStore interface {
	GetPfxCertificate(secretKey string) []byte
	convertSecret(secretKey string, secret *v1.Secret) bool
	eraseSecret(secretKey string)
}

type secretStore struct {
	conversionSync sync.Mutex
	Cache          cache.ThreadSafeStore
}

// NewSecretStore creates a new Secret Store object
func NewSecretStore() SecretStore {
	return &secretStore{
		Cache: cache.NewThreadSafeStore(cache.Indexers{}, cache.Indices{}),
	}
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

func (s *secretStore) GetPfxCertificate(secretKey string) []byte {
	certInterface, exists := s.Cache.Get(secretKey)
	if exists {
		if cert, ok := certInterface.([]byte); ok {
			return cert
		}
	}
	return nil
}

func (s *secretStore) eraseSecret(secretKey string) {
	s.conversionSync.Lock()
	defer s.conversionSync.Unlock()

	s.Cache.Delete(secretKey)
}

func (s *secretStore) convertSecret(secretKey string, secret *v1.Secret) bool {
	s.conversionSync.Lock()
	defer s.conversionSync.Unlock()

	// check if this is a secret with the correct type
	if secret.Type != "kubernetes.io/tls" {
		glog.Errorf("secret [%v] is not type kubernetes.io/tls", secretKey)
		return false
	}

	if len(secret.Data["tls.key"]) == 0 || len(secret.Data["tls.crt"]) == 0 {
		glog.Errorf("secret [%v] is malformed, tls.key or tls.crt is not defined", secretKey)
		return false
	}

	tempfileCert, err := ioutil.TempFile("", "appgw-ingress-cert")
	if err != nil {
		glog.Error("unable to create temporary file for certificate conversion")
		return false
	}
	defer os.Remove(tempfileCert.Name())

	tempfileKey, err := ioutil.TempFile("", "appgw-ingress-key")
	if err != nil {
		glog.Error("unable to create temporary file for certificate conversion")
		return false
	}
	defer os.Remove(tempfileKey.Name())

	if err := writeFileDecode(secret.Data["tls.crt"], tempfileCert); err != nil {
		glog.Errorf("unable to write secret [%v].tls.crt to temporary file, error: %v", secretKey, err)
		return false
	}

	if err := writeFileDecode(secret.Data["tls.key"], tempfileKey); err != nil {
		glog.Errorf("unable to write secret [%v].tls.key to temporary file, error: %v", secretKey, err)
		return false
	}

	// both cert and key are in temp file now, call openssl
	var cout, cerr bytes.Buffer
	cmd := exec.Command("openssl", "pkcs12", "-export", "-in", tempfileCert.Name(), "-inkey", tempfileKey.Name(), "-password", "pass:msazure")
	cmd.Stderr = &cerr
	cmd.Stdout = &cout

	// if openssl exited with an error or the output is empty, report error
	if err := cmd.Run(); err != nil || len(cout.Bytes()) == 0 {
		glog.Errorf("unable to export using openssl, error=[%v], stderr=[%v]", err, cerr.String())
		return false
	}

	pfxCert := cout.Bytes()

	glog.V(1).Infof("converted secret [%v]", secretKey)
	// TODO i'm not sure if comparison against existing certificate can help
	// us optimize by eliminating some events
	_, exists := s.Cache.Get(secretKey)
	if exists {
		s.Cache.Update(secretKey, pfxCert)
	} else {
		s.Cache.Add(secretKey, pfxCert)
	}

	return true
}
