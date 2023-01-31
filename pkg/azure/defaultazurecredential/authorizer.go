package defaultazurecredential

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"
	"k8s.io/klog/v2"
)

// NewAuthorizer returns an autorest.Authorizer that uses the DefaultAzureCredential
// from the Azure SDK for Go.
//
// DefaultAzureCredential uses the following credential types in order in version 1.3 or higher.
//   - [EnvironmentCredential]
//   - [WorkloadIdentityCredential], if environment variable configuration is set by the Azure workload
//     identity webhook. Use [WorkloadIdentityCredential] directly when not using the webhook or needing
//     more control over its configuration.
//   - [ManagedIdentityCredential]
//   - [AzureCLICredential]
func NewAuthorizer() (autorest.Authorizer, error) {
	cred, err := azidentity.NewDefaultAzureCredential(&azidentity.DefaultAzureCredentialOptions{})
	if err != nil {
		return nil, err
	}

	return autorest.NewBearerAuthorizer(&tokenCredentialWrapper{
		cred: cred,
	}), nil
}

type tokenCredentialWrapper struct {
	cred azcore.TokenCredential
}

func (w *tokenCredentialWrapper) OAuthToken() string {
	klog.V(7).Info("Getting Azure token using DefaultAzureCredential")

	token, err := w.cred.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})

	if err != nil {
		klog.Error("Error getting Azure token: ", err)
		return ""
	}

	return token.Token
}
