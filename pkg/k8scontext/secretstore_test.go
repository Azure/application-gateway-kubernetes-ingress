// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = ginkgo.Describe("Testing K8sContext.SecretStore", func() {
	secretsStore := NewSecretStore()
	ginkgo.Context("Test ConvertSecret function", func() {
		secret := v1.Secret{}
		ginkgo.It("Should have returned an error - unrecognized type of secret", func() {
			err := secretsStore.ConvertSecret("someKey", &secret)
			Expect(err).To(Equal(ErrorUnknownSecretType))
		})
		ginkgo.It("", func() {
			malformed := secret
			malformed.Type = recognizedSecretType
			err := secretsStore.ConvertSecret("someKey", &malformed)
			Expect(err).To(Equal(ErrorMalformedSecret))
		})
		ginkgo.It("", func() {
			malformed := secret
			malformed.Type = recognizedSecretType
			malformed.Data = make(map[string][]byte)
			malformed.Data[tlsKey] = []byte("X")
			malformed.Data[tlsCrt] = []byte("Y")
			err := secretsStore.ConvertSecret("someKey", &malformed)
			Expect(err).To(Equal(ErrorExportingWithOpenSSL))
		})
		ginkgo.It("", func() {
			err := secretsStore.ConvertSecret("someKey", tests.NewSecretTestFixture())
			Expect(err).ToNot(HaveOccurred())
			actual := secretsStore.GetPfxCertificate("someKey")
			Expect(len(actual)).To(Equal(2477))
		})
	})
})
