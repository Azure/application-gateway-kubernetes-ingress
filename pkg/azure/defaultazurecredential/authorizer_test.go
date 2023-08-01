package defaultazurecredential

import (
	"os"
	"testing"
)

func TestTokenScopeFromEnvironment(t *testing.T) {
	scope := map[string]string{
		"AZUREPUBLICCLOUD":       "https://management.azure.com/.default",
		"AZURECHINACLOUD":        "https://management.chinacloudapi.cn/.default",
		"AZUREUSGOVERNMENTCLOUD": "https://management.usgovcloudapi.net/.default",
	}

	for env, expectedScope := range scope {
		os.Setenv("AZURE_ENVIRONMENT", env)
		scope := tokenScopeFromEnvironment()
		if scope != expectedScope {
			t.Errorf("Expected scope %s, got %s", expectedScope, scope)
		}
	}
}
