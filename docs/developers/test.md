# Testing the controller

* [Unit Tests](#unit-tests)
* [E2E Tests](#e2e-tests)
* [Testing Tips](#testing-tips)

## Unit Tests
As is the convention in go, unit tests for the `.go` file you want to test live in the same folder and end with `_test.go`.
We use the `ginkgo`/`gomega`  testing framework for writing the tests.

To execute the tests, use
```bash
go test -v -tags unittest ./...
```

## E2E Tests
E2E tests are going to test the specific scenarios with a real AKS and App Gateway setup with AGIC installed on it.

E2E tests are automatically run every day 3 AM in the morning using an [E2E pipeline](https://dev.azure.com/azure/application-gateway-kubernetes-ingress/_release?_a=releases&view=mine&definitionId=14).

If you have cluster with AGIC installed, you can run e2e tests simply by:
```bash
go test -v -tags e2e ./...
```

You can also execute the `run-e2e.sh` which is used in the E2E pipeline to invoke the tests. This script will install AGIC with the version provided.
```bash
export version="<agic-version>"
export applicationGatewayId="<resource-id>"
export identityResourceId="<agic-identity-resource-id>"
export identityClientId="<agic-identity-client-id>"

./scripts/e2e/run-e2e.sh
```

## Testing Tips
* If you just want to run a specific set of tests, then an easy way is add `F` (Focus) to the `It`, `Context`, `Describe` directive in the test.
    For example:
    ```go
    FContext("Test obtaining a single certificate for an existing host", func() {
        cb := newConfigBuilderFixture(nil)
        ingress := tests.NewIngressFixture()
        hostnameSecretIDMap := cb.newHostToSecretMap(ingress)
        actualSecret, actualSecretID := cb.getCertificate(ingress, host1, hostnameSecretIDMap)

        It("should have generated the expected secret", func() {
            Expect(*actualSecret).To(Equal("eHl6"))
        })

        It("should have generated the correct secretID struct", func() {
            Expect(*actualSecretID).To(Equal(expectedSecret))
        })
    })
    ```