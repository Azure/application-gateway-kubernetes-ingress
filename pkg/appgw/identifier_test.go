package appgw

import (
	"fmt"
	"testing"
)

func TestRedirectConfigurationID(t *testing.T) {
	ident := Identifier{
		SubscriptionID: "subs",
		ResourceGroup: "rg",
		AppGwName: "appgwname",
	}
	actual := ident.redirectConfigurationID("cofiguration-name")
	expected := "/subscriptions/subs/resourceGroups/rg/providers/" +
		"Microsoft.Network/applicationGateways/appgwname" +
		"/redirectConfigurations/cofiguration-name"
	if actual != expected {
		t.Error(fmt.Sprintf("\nExpected %s\nActually %s", expected, actual))
	}
}
