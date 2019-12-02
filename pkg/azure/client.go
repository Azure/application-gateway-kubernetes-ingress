// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
	r "github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/golang/glog"
)

// AzClient is an interface for client to Azure
type AzClient interface {
	GetGateway() (n.ApplicationGateway, error)
	UpdateGateway(*n.ApplicationGateway) error
	DeployGatewayWithVnet(ResourceGroup, ResourceName, ResourceName, string) error
	DeployGatewayWithSubnet(string) error

	GetPublicIP(string) (n.PublicIPAddress, error)
}

type azClient struct {
	appGatewaysClient     n.ApplicationGatewaysClient
	publicIPsClient       n.PublicIPAddressesClient
	virtualNetworksClient n.VirtualNetworksClient
	subnetsClient         n.SubnetsClient
	groupsClient          r.GroupsClient
	deploymentsClient     r.DeploymentsClient
	authorizer            autorest.Authorizer

	subscriptionID    SubscriptionID
	resourceGroupName ResourceGroup
	appGwName         ResourceName
	memoizedIPs       map[string]n.PublicIPAddress

	ctx context.Context
}

// NewAzClient returns an Azure Client
func NewAzClient(subscriptionID SubscriptionID, resourceGroupName ResourceGroup, appGwName ResourceName, authorizer autorest.Authorizer) AzClient {
	settings, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		return nil
	}

	userAgent := fmt.Sprintf("ingress-appgw/%s", version.Version)
	az := &azClient{
		appGatewaysClient:     n.NewApplicationGatewaysClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, string(subscriptionID)),
		publicIPsClient:       n.NewPublicIPAddressesClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, string(subscriptionID)),
		virtualNetworksClient: n.NewVirtualNetworksClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, string(subscriptionID)),
		subnetsClient:         n.NewSubnetsClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, string(subscriptionID)),
		groupsClient:          r.NewGroupsClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, string(subscriptionID)),
		deploymentsClient:     r.NewDeploymentsClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, string(subscriptionID)),

		subscriptionID:    subscriptionID,
		resourceGroupName: resourceGroupName,
		appGwName:         appGwName,
		memoizedIPs:       make(map[string]n.PublicIPAddress),

		ctx:        context.Background(),
		authorizer: authorizer,
	}

	if err := az.appGatewaysClient.AddToUserAgent(userAgent); err != nil {
		glog.Error("Error adding User Agent to App Gateway client: ", userAgent)
	}
	az.appGatewaysClient.Authorizer = az.authorizer

	if err := az.publicIPsClient.AddToUserAgent(userAgent); err != nil {
		glog.Error("Error adding User Agent to Public IP client: ", userAgent)
	}
	az.publicIPsClient.Authorizer = az.authorizer

	if err := az.virtualNetworksClient.AddToUserAgent(userAgent); err != nil {
		glog.Error("Error adding User Agent to Virtual Networks client: ", userAgent)
	}
	az.virtualNetworksClient.Authorizer = az.authorizer

	if err := az.subnetsClient.AddToUserAgent(userAgent); err != nil {
		glog.Error("Error adding User Agent to Subnets client: ", userAgent)
	}
	az.subnetsClient.Authorizer = az.authorizer

	if err := az.groupsClient.AddToUserAgent(userAgent); err != nil {
		glog.Error("Error adding User Agent to Groups client: ", userAgent)
	}
	az.groupsClient.Authorizer = az.authorizer

	if err := az.deploymentsClient.AddToUserAgent(userAgent); err != nil {
		glog.Error("Error adding User Agent to Deployments client: ", userAgent)
	}
	az.deploymentsClient.Authorizer = az.authorizer

	return az
}

func (az *azClient) GetGateway() (n.ApplicationGateway, error) {
	return az.appGatewaysClient.Get(az.ctx, string(az.resourceGroupName), string(az.appGwName))
}

func (az *azClient) UpdateGateway(appGwObj *n.ApplicationGateway) (err error) {
	appGwFuture, err := az.appGatewaysClient.CreateOrUpdate(az.ctx, string(az.resourceGroupName), string(az.appGwName), *appGwObj)
	if err != nil {
		return
	}

	// Wait until deployment finshes and save the error message
	err = appGwFuture.WaitForCompletionRef(az.ctx, az.appGatewaysClient.BaseClient.Client)
	return
}

func (az *azClient) GetPublicIP(resourceID string) (n.PublicIPAddress, error) {
	if ip, ok := az.memoizedIPs[resourceID]; ok {
		return ip, nil
	}

	_, resourceGroupName, publicIPName := ParseResourceID(resourceID)

	ip, err := az.publicIPsClient.Get(az.ctx, string(resourceGroupName), string(publicIPName), "")
	if err != nil {
		return n.PublicIPAddress{}, err
	}
	az.memoizedIPs[resourceID] = ip
	return ip, nil
}

// DeployGateway is a method that deploy the appgw and related resources
func (az *azClient) DeployGatewayWithVnet(resourceGroupName ResourceGroup, vnetName ResourceName, subnetName ResourceName, subnetPrefix string) (err error) {
	vnet, err := az.getVnet(resourceGroupName, vnetName)
	if err != nil {
		return
	}

	glog.Infof("Checking the Vnet %s for a subnet with prefix %s", vnetName, subnetPrefix)
	subnet, err := az.findSubnet(vnet, subnetName, subnetPrefix)
	if err != nil {
		if subnetPrefix == "" {
			glog.Infof("Unable to find a subnet with subnetName %s. Please provide subnetPrefix in order to allow AGIC to create a subnet in Vnet %s", subnetName, vnetName)
			return
		}

		glog.Infof("Unable to find a subnet. Creating a subnet %s with prefix %s in Vnet %s", subnetName, subnetPrefix, vnetName)
		subnet, err = az.createSubnet(vnet, subnetName, subnetPrefix)
		if err != nil {
			return
		}
	}

	err = az.DeployGatewayWithSubnet(*subnet.ID)
	return
}

// DeployGateway is a method that deploy the appgw and related resources
func (az *azClient) DeployGatewayWithSubnet(subnetID string) (err error) {
	glog.Infof("Deploying Gateway")

	// Check if group exists
	group, err := az.getGroup()
	if err != nil {
		return
	}
	glog.Infof("Using resource group: %v", *group.Name)

	deploymentName := string(az.appGwName)
	glog.Infof("Starting ARM template deployment: %s", deploymentName)
	result, err := az.createDeployment(subnetID)
	if err != nil {
		return
	}
	if result.Name != nil {
		glog.Infof("Completed deployment %v: %v", deploymentName, *result.Properties.ProvisioningState)
	} else {
		glog.Infof("Completed deployment %v (no data returned to SDK)", deploymentName)
	}

	return
}

// Create a resource group for the deployment.
func (az *azClient) getGroup() (r.Group, error) {
	return az.groupsClient.Get(az.ctx, string(az.resourceGroupName))
}

func (az *azClient) getVnet(resourceGroupName ResourceGroup, vnetName ResourceName) (n.VirtualNetwork, error) {
	return az.virtualNetworksClient.Get(az.ctx, string(resourceGroupName), string(vnetName), "")
}

func (az *azClient) findSubnet(vnet n.VirtualNetwork, subnetName ResourceName, subnetPrefix string) (subnet n.Subnet, err error) {
	for _, subnet := range *vnet.Subnets {
		if string(subnetName) == *subnet.Name && (subnetPrefix == "" || subnetPrefix == *subnet.AddressPrefix) {
			return subnet, nil
		}
	}
	err = errors.New("Unable to find subnet with matching subnetName and subnetPrefix")
	return
}

func (az *azClient) createSubnet(vnet n.VirtualNetwork, subnetName ResourceName, subnetPrefix string) (subnet n.Subnet, err error) {
	_, resourceGroup, vnetName := ParseResourceID(*vnet.ID)
	subnet = n.Subnet{
		SubnetPropertiesFormat: &n.SubnetPropertiesFormat{
			AddressPrefix: &subnetPrefix,
		},
	}
	subnetFuture, err := az.subnetsClient.CreateOrUpdate(az.ctx, string(resourceGroup), string(vnetName), string(subnetName), subnet)
	if err != nil {
		return
	}

	// Wait until deployment finshes and save the error message
	err = subnetFuture.WaitForCompletionRef(az.ctx, az.subnetsClient.BaseClient.Client)
	if err != nil {
		return
	}

	return az.subnetsClient.Get(az.ctx, string(resourceGroup), string(vnetName), string(subnetName), "")
}

// Create the deployment
func (az *azClient) createDeployment(subnetID string) (deployment r.DeploymentExtended, err error) {
	template := getTemplate()
	if err != nil {
		return
	}
	params := map[string]interface{}{
		"applicationGatewayName": map[string]string{
			"value": string(az.appGwName),
		},
		"applicationGatewaySubnetId": map[string]string{
			"value": subnetID,
		},
	}

	deploymentFuture, err := az.deploymentsClient.CreateOrUpdate(
		az.ctx,
		string(az.resourceGroupName),
		string(az.appGwName),
		r.Deployment{
			Properties: &r.DeploymentProperties{
				Template:   template,
				Parameters: params,
				Mode:       r.Incremental,
			},
		},
	)
	if err != nil {
		return
	}
	err = deploymentFuture.Future.WaitForCompletionRef(az.ctx, az.deploymentsClient.BaseClient.Client)
	if err != nil {
		return
	}
	return deploymentFuture.Result(az.deploymentsClient)
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
