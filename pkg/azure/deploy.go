// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import (
	"context"
	"encoding/json"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	"github.com/Azure/go-autorest/autorest"
)

var ctx = context.Background()

// Deploy is a method that deploy the appge and related resources
func Deploy(subscriptionID SubscriptionID, resourceGroupName ResourceGroup, appgwName ResourceName, subnetID string, authorizer autorest.Authorizer) (err error) {
	group, err := getGroup(subscriptionID, resourceGroupName, authorizer)
	if err != nil {
		return
	}
	log.Printf("Created group: %v", *group.Name)

	deploymentName := string(appgwName)
	log.Printf("Starting deployment: %s", deploymentName)
	result, err := createDeployment(subscriptionID, resourceGroupName, appgwName, subnetID, authorizer)
	if err != nil {
		return
	}
	if result.Name != nil {
		log.Printf("Completed deployment %v: %v", deploymentName, *result.Properties.ProvisioningState)
	} else {
		log.Printf("Completed deployment %v (no data returned to SDK)", deploymentName)
	}

	return
}

// Create a resource group for the deployment.
func getGroup(subscriptionID SubscriptionID, resourceGroupName ResourceGroup, authorizer autorest.Authorizer) (resources.Group, error) {
	groupsClient := resources.NewGroupsClient(string(subscriptionID))
	groupsClient.Authorizer = authorizer

	return groupsClient.Get(ctx, string(resourceGroupName))
}

// Create the deployment
func createDeployment(subscriptionID SubscriptionID, resourceGroupName ResourceGroup, appgwName ResourceName, subnetID string, authorizer autorest.Authorizer) (deployment resources.DeploymentExtended, err error) {
	template := getTemplate()
	if err != nil {
		return
	}
	params := map[string]interface{}{
		"applicationGatewayName": map[string]string{
			"value": string(appgwName),
		},
		"applicationGatewaySubnetId": map[string]string{
			"value": subnetID,
		},
	}

	deploymentsClient := resources.NewDeploymentsClient(string(subscriptionID))
	deploymentsClient.Authorizer = authorizer

	deploymentFuture, err := deploymentsClient.CreateOrUpdate(
		ctx,
		string(resourceGroupName),
		string(appgwName),
		resources.Deployment{
			Properties: &resources.DeploymentProperties{
				Template:   template,
				Parameters: params,
				Mode:       resources.Incremental,
			},
		},
	)
	if err != nil {
		return
	}
	err = deploymentFuture.Future.WaitForCompletionRef(ctx, deploymentsClient.BaseClient.Client)
	if err != nil {
		return
	}
	return deploymentFuture.Result(deploymentsClient)
}

func getTemplate() map[string]interface{} {
	template := `
	{
		"$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		"contentVersion": "1.0.0.0",
		"parameters": {
			"applicationGatewayName": {
				"defaultValue": "appgw",
				"type": "string",
				"metadata": {
					"description": "Name of the Application Gateway."
				}
			},
			"applicationGatewaySubnetId": {
				"type": "string",
				"metadata": {
					"description": "Resource Id of Subnet in which Application Gateway will be deployed."
				}
			},
			"applicationGatewaySku": {
				"defaultValue": "Standard_v2",
				"allowedValues": [
					"Standard_v2",
					"WAF_v2"
				],
				"type": "string",
				"metadata": {
					"description": "The sku of the Application Gateway. Default: WAF_v2 (Detection mode). In order to further customize WAF, use azure portal or cli."
				}
			}
		},
		"variables": {
			"resgpguid": "[substring(replace(guid(resourceGroup().id), '-', ''), 0, 4)]",
			"vnetName": "[concat('virtualnetwork' , variables('resgpguid'))]",
			"applicationGatewayPublicIpName": "[concat(parameters('applicationGatewayName'), '-appgwpip')]",
			"applicationGatewayPublicIpId": "[resourceId('Microsoft.Network/publicIPAddresses',variables('applicationGatewayPublicIpName'))]",
			"applicationGatewayId": "[resourceId('Microsoft.Network/applicationGateways', parameters('applicationGatewayName'))]",
			"webApplicationFirewallConfiguration": {
			  "enabled": "true",
			  "firewallMode": "Detection"
			}
		},
		"resources": [
			{
				"type": "Microsoft.Network/publicIPAddresses",
				"name": "[variables('applicationGatewayPublicIpName')]",
				"apiVersion": "2018-08-01",
				"location": "[resourceGroup().location]",
				"sku": {
					"name": "Standard"
				},
				"properties": {
					"publicIPAllocationMethod": "Static"
				}
			},
			{
				"type": "Microsoft.Network/applicationGateways",
				"name": "[parameters('applicationGatewayName')]",
				"apiVersion": "2018-08-01",
				"location": "[resourceGroup().location]",
				"tags": {
					"managed-by-k8s-ingress": "true"
				},
				"properties": {
					"sku": {
						"name": "[parameters('applicationGatewaySku')]",
						"tier": "[parameters('applicationGatewaySku')]",
						"capacity": 2
					},
					"gatewayIPConfigurations": [
						{
							"name": "appGatewayIpConfig",
							"properties": {
								"subnet": {
									"id": "[parameters('applicationGatewaySubnetId')]"
								}
							}
						}
					],
					"frontendIPConfigurations": [
						{
							"name": "appGatewayFrontendIP",
							"properties": {
								"PublicIPAddress": {
									"id": "[variables('applicationGatewayPublicIpId')]"
								}
							}
						}
					],
					"frontendPorts": [
						{
							"name": "httpPort",
							"properties": {
								"Port": 80
							}
						},
						{
							"name": "httpsPort",
							"properties": {
								"Port": 443
							}
						}
					],
					"backendAddressPools": [
						{
							"name": "bepool",
							"properties": {
								"backendAddresses": []
							}
						}
					],
					"httpListeners": [
						{
							"name": "httpListener",
							"properties": {
								"protocol": "Http",
								"frontendPort": {
									"id": "[concat(variables('applicationGatewayId'), '/frontendPorts/httpPort')]"
								},
								"frontendIPConfiguration": {
									"id": "[concat(variables('applicationGatewayId'), '/frontendIPConfigurations/appGatewayFrontendIP')]"
								}
							}
						}
					],
					"backendHttpSettingsCollection": [
						{
							"name": "setting",
							"properties": {
								"port": 80,
								"protocol": "Http"
							}
						}
					],
					"requestRoutingRules": [
						{
							"name": "rule1",
							"properties": {
								"httpListener": {
									"id": "[concat(variables('applicationGatewayId'), '/httpListeners/httpListener')]"
								},
								"backendAddressPool": {
									"id": "[concat(variables('applicationGatewayId'), '/backendAddressPools/bepool')]"
								},
								"backendHttpSettings": {
									"id": "[concat(variables('applicationGatewayId'), '/backendHttpSettingsCollection/setting')]"
								}
							}
						}
					],
					"webApplicationFirewallConfiguration": "[if(equals(parameters('applicationGatewaySku'), 'WAF_v2'), variables('webApplicationFirewallConfiguration'), json('null'))]"
				},
				"dependsOn": [
					"[concat('Microsoft.Network/publicIPAddresses/', variables('applicationGatewayPublicIpName'))]"
				]
			}
		],
		"outputs": {
			"subscriptionId": {
				"type": "string",
				"value": "[subscription().subscriptionId]"
			},
			"resourceGroupName": {
				"type": "string",
				"value": "[resourceGroup().name]"
			},
			"applicationGatewayName": {
				"type": "string",
				"value": "[parameters('applicationGatewayName')]"
			}
		}
	}`

	contents := make(map[string]interface{})
	json.Unmarshal([]byte(template), &contents)
	return contents
}
