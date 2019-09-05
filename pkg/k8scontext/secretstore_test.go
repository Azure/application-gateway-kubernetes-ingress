// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = ginkgo.Describe("Testing K8sContext.SecretStore", func() {
	secretsStore := NewSecretStore()
	ginkgo.Context("Test convertSecret function", func() {
		secret := v1.Secret{}
		ginkgo.It("Should have returned an error - unrecognized type of secret", func() {
			err := secretsStore.convertSecret("someKey", &secret)
			Expect(err).To(Equal(ErrorUnknownSecretType))
		})
		ginkgo.It("", func() {
			malformed := secret
			malformed.Type = recognizedSecretType
			err := secretsStore.convertSecret("someKey", &malformed)
			Expect(err).To(Equal(ErrorMalformedSecret))
		})
		ginkgo.It("", func() {
			malformed := secret
			malformed.Type = recognizedSecretType
			malformed.Data = make(map[string][]byte)
			malformed.Data[tlsKey] = []byte("X")
			malformed.Data[tlsCrt] = []byte("Y")
			err := secretsStore.convertSecret("someKey", &malformed)
			Expect(err).To(Equal(ErrorExportingWithOpenSSL))
		})
		ginkgo.It("", func() {
			goodSecret := secret
			goodSecret.Type = recognizedSecretType
			goodSecret.Data = make(map[string][]byte)
			goodSecret.Data[tlsKey] = []byte("-----BEGIN PRIVATE KEY-----\n" +
				"MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDFEs4opOIMHYna\n" +
				"wMio1JHZaQWDZEP8fsL23Rozhow0vVokthPk4wGBKYpc8XYBbWFs5pUuExeOjeRW\n" +
				"jdNArwn5jCZYaxtqfdrj2kLHFHPCTwmbzn+qvPkvp/ZJyeY+4eIe7soGzO6hoj/w\n" +
				"HHdry7rPiap5R5EMfHzfl1TZ5WfixqnxKVEc33VRD9xwQIHwJTGnoI2bTGK3vK5q\n" +
				"90Glxyc4FAqo6xBguo6ZfOCqPYHXAKtaMj5hcr2dA0/7rJ/xNthDdnQwhETU2BgQ\n" +
				"9PvqfuMif+r/VM4KmYjQYu6+NN8VDVq6eSx4dxIzqWZ/NdSeoIri+6Gpa0AncMrq\n" +
				"3t7OjuQjAgMBAAECggEAKfzOtbQjgSdN9rB6UBYyGNsaVJspLQOo8EW9TlsNRjNN\n" +
				"oGK2rF59NJKwKws69CTky/n4sL9aloG+s342EyL4AhYNGWuAhNjZqRAYiCfgXfbO\n" +
				"+kYtxyfKA5BKlgARMTaZIbQIkRhag095ReQawXm/jHYtPvezfLCNPmoUpvQMhTEk\n" +
				"jzhhB7Ao5JPkw6jjnYa4raETYR3LTdFwhfU1WecEJ+Mj1hGX8ANC8cdHYxvkomcl\n" +
				"/ucl99siNJKYHZ6wWXpLRICZyTyLCcCnICj2g/+8BiV9pokrUHYW5diLDN4UBHnQ\n" +
				"Qe2LZnC+hIU8Vvq2z9Wy8tF8Z2LMmswK+kIff7tNuQKBgQDxil1AaMSCAbTecErf\n" +
				"RJkK81YvtMvM5ha2lhHOdnGvl/aVMdQ1rAkklGXMbIz/e87gOR3PfbmY67QR9aEz\n" +
				"CTXjfWG6J5Ri99kEn3af3AOrbJ6dbgaZWKwvtuDfXciuFo+0K5eQcC6r5Cr5Wjs3\n" +
				"DAnYWMGz9sUyMg0s6OWqifLMjQKBgQDQ3vqOXVzjRbvhyq/QWerl/x3jUIs/fY3e\n" +
				"6IAcf7jyihvQWAX361yjQig8n8D6XcPo1GKmKr93ra7cVdH3eQ1ICG9KzfMyG0PQ\n" +
				"H6qkft19BAwsCK687LeotTGX4qXavXG9AP8tLyq8WRGdmQPStpses7oUjOBk3rKo\n" +
				"8puKExe/bwKBgQDAupvf2fj6l2v/lXBYqH7JexLJLCT2EJ4NAL+ik2XxK3tI3qKq\n" +
				"VORSuMpljDQRY3PV/B0qQ/KE74YWUn1WoMHMDG6fQBepxIP4qVjZA5A2B4ykp3dC\n" +
				"gruZsv3JnSaUqlHt/F6KlMjYxU34+yOGr+dnJqMg+wWsIL3cmNUw97OxvQKBgCo4\n" +
				"6O1ecih/MDu0fVXg11sm9yO8ZGmxN7yXw03/g6ODx5uWL56uNUvLU9btdFUoHzIx\n" +
				"vL9aZNoMggyITKl6DvVAvz6f40l9uXeY7yXRf3SGHO/J0YjfUUEJX70UU/Kj2Rob\n" +
				"2XmIz1rDpov1IpC12SWbr0H4OGQroHIGmOqQcXyBAoGAMFJzs9K/bx6MA9lZW8Vw\n" +
				"adbuUcqFjOAryk8fNzZNBgYADRHfNz1Az3vqIi1zWcnimou0M4o2BRGUc805wy7V\n" +
				"YfkIyRQ5bIIVGNpP19dEOSsJ8pYAr+Bo/3GjXxUe6O6PxF3hbfPJNWt11refYC27\n" +
				"dZsRsRJX4pAw+BznAZodf6Q=\n" +
				"-----END PRIVATE KEY-----\n")
			goodSecret.Data[tlsCrt] = []byte("-----BEGIN CERTIFICATE-----\n" +
				"MIIDazCCAlOgAwIBAgIUOX75BZ3gP92zRT89ZO34HXdi44QwDQYJKoZIhvcNAQEL\n" +
				"BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM\n" +
				"GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0xOTA4MjkwMDQ0NDdaFw0yMDA4\n" +
				"MjgwMDQ0NDdaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw\n" +
				"HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEB\n" +
				"AQUAA4IBDwAwggEKAoIBAQDFEs4opOIMHYnawMio1JHZaQWDZEP8fsL23Rozhow0\n" +
				"vVokthPk4wGBKYpc8XYBbWFs5pUuExeOjeRWjdNArwn5jCZYaxtqfdrj2kLHFHPC\n" +
				"Twmbzn+qvPkvp/ZJyeY+4eIe7soGzO6hoj/wHHdry7rPiap5R5EMfHzfl1TZ5Wfi\n" +
				"xqnxKVEc33VRD9xwQIHwJTGnoI2bTGK3vK5q90Glxyc4FAqo6xBguo6ZfOCqPYHX\n" +
				"AKtaMj5hcr2dA0/7rJ/xNthDdnQwhETU2BgQ9PvqfuMif+r/VM4KmYjQYu6+NN8V\n" +
				"DVq6eSx4dxIzqWZ/NdSeoIri+6Gpa0AncMrq3t7OjuQjAgMBAAGjUzBRMB0GA1Ud\n" +
				"DgQWBBTCTeqqryPyXKMAoo28CGKvS2dvuDAfBgNVHSMEGDAWgBTCTeqqryPyXKMA\n" +
				"oo28CGKvS2dvuDAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQCp\n" +
				"e7uP6D0bU6Z/ZuWZrZUwvo054912wg7O7zNJeZ1dnV9M/3ozR5UR1LSilhRgtOLD\n" +
				"mUIQtQdoJCTnPb/FrD7ZvOL5e0CjbvKSs7UxhvsOBiE4EQCHS4Gp1HUtFRS+H60U\n" +
				"Z0cUG4CnbjJy0JmXpEq+B1McDc7QtR9p0JJiOIJN59255u/Kdg+0NWdRsB6zdZMn\n" +
				"p4gifcw3N8eErYFSs6mHhblTOROMf0kCGan6qyx08Lk/t3YI33ZAktk8T5GVSe3A\n" +
				"o1nu88fKxKLEH6kcBzx35dt3CmMsHCXgX58R+OHD8boJteLkkuc+h+mzO7G8h/Bv\n" +
				"LloWsUALcQTN0LMl33F8\n" +
				"-----END CERTIFICATE-----\n")
			err := secretsStore.convertSecret("someKey", &goodSecret)
			Expect(err).ToNot(HaveOccurred())
			actual := secretsStore.GetPfxCertificate("someKey")
			Expect(len(actual)).To(Equal(2477))
		})
	})
})
