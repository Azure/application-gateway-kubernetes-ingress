package defaultazurecredential

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
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

	scope := tokenScopeFromEnvironment()
	klog.V(7).Infof("Fetching token with scope %s", scope)
	return autorest.NewBearerAuthorizer(&tokenCredentialWrapper{
		cred:  cred,
		scope: scope,
	}), nil
}

func tokenScopeFromEnvironment() string {
	cloud := os.Getenv("AZURE_ENVIRONMENT")
	env, err := azure.EnvironmentFromName(cloud)
	if err != nil {
		env = azure.PublicCloud
	}

	return fmt.Sprintf("%s.default", env.TokenAudience)
}

type tokenCredentialWrapper struct {
	cred  azcore.TokenCredential
	scope string
}

func (w *tokenCredentialWrapper) OAuthToken() string {
	klog.V(7).Info("Getting Azure token using DefaultAzureCredential")

	token, err := w.cred.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{w.scope},
	})

	if err != nil {
		klog.Error("Error getting Azure token: ", err)
		return ""
	}

	return token.Token
}
