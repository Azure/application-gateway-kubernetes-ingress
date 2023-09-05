// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

var _ = ginkgo.Describe("Testing K8sContext.SecretStore", func() {
	secretsStore := NewSecretStore(nil)

	ginkgo.DescribeTable("when converting certificate to PFX",
		func(secret *v1.Secret, expectedError controllererrors.ErrorCode) {
			err := secretsStore.ConvertSecret("someKey", secret)
			Expect(err.(*controllererrors.Error).Code).To(Equal(expectedError))
		},
		ginkgo.Entry("no type in secret", &v1.Secret{}, controllererrors.ErrorUnknownSecretType),
		ginkgo.Entry("unrecognized type of secret", &v1.Secret{Type: v1.SecretTypeOpaque}, controllererrors.ErrorUnknownSecretType),
		ginkgo.Entry("malformed data", &v1.Secret{Type: v1.SecretTypeTLS, Data: map[string][]byte{}}, controllererrors.ErrorMalformedSecret),
		ginkgo.Entry("invalid data", &v1.Secret{Type: v1.SecretTypeTLS, Data: map[string][]byte{
			v1.TLSCertKey:       []byte("X"),
			v1.TLSPrivateKeyKey: []byte("X"),
		}}, controllererrors.ErrorExportingWithOpenSSL),
	)

	ginkgo.When("certificate gets stored", func() {
		ginkgo.It("should be retrivable with the secret key", func() {
			err := secretsStore.ConvertSecret("someKey", tests.NewSecretTestFixture())
			Expect(err).ToNot(HaveOccurred())
			actual := secretsStore.GetPfxCertificate("someKey")
			Expect(len(actual)).To(BeNumerically(">", 0))
		})
	})

	ginkgo.When("certificate is no cached", func() {
		ginkgo.It("should get it from the api-server", func() {
			secret := tests.NewSecretTestFixture()
			var client kubernetes.Interface = testclient.NewSimpleClientset(secret)
			secretsStore := NewSecretStore(client)

			actual := secretsStore.GetPfxCertificate(secret.Namespace + "/" + secret.Name)
			Expect(len(actual)).To(BeNumerically(">", 0))
		})
	})
})
